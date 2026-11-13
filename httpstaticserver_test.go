package main

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHTTPStaticServerSetsContentSecurityPolicy(t *testing.T) {
	root, err := os.MkdirTemp("", "gohttpserver-csp-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	server := NewHTTPStaticServer(root, true)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	server.ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Content-Security-Policy"); got != contentSecurityPolicy {
		t.Fatalf("Content-Security-Policy header = %q, want %q", got, contentSecurityPolicy)
	}
}

func TestReadmePreviewShowdownDisablesHTML(t *testing.T) {
	data, err := os.ReadFile("assets/js/index.js")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "noHTML: true") {
		t.Fatal("showdown converter must disable raw HTML passthrough with noHTML: true")
	}
}

func TestReadmePreviewTemplateEscapesContent(t *testing.T) {
	data, err := os.ReadFile("assets/index.html")
	if err != nil {
		t.Fatal(err)
	}
	template := string(data)

	if strings.Contains(template, "{{{preview.contentHTML") || strings.Contains(template, "{{{ preview.contentHTML") {
		t.Fatal("README preview must not use Vue raw-HTML triple curly interpolation")
	}
	if !strings.Contains(template, "{{ preview.contentHTML }}") {
		t.Fatal("README preview must use escaped Vue double curly interpolation")
	}
}

func newUploadRequest(t *testing.T, url, filename string, content []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func listUploadTempFiles(t *testing.T, dir string) []string {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(dir, ".upload-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

// TestUploadIsAtomicForConcurrentDownloads reproduces the empty/truncated
// downloads reported in issue #219: while a file is being uploaded, concurrent
// readers must see either the old content or the new one, never a partial file.
func TestUploadIsAtomicForConcurrentDownloads(t *testing.T) {
	root := t.TempDir()
	server := NewHTTPStaticServer(root, true)
	server.Upload = true
	ts := httptest.NewServer(server)
	defer ts.Close()

	const size = 8 << 20
	oldContent := bytes.Repeat([]byte("a"), size)
	newContent := bytes.Repeat([]byte("b"), size)

	dstPath := filepath.Join(root, "upload.json")
	if err := os.WriteFile(dstPath, oldContent, 0644); err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	targetURL := ts.URL + "/upload.json"
	// Uploads are posted to the destination directory and named via the form.
	uploadReq := newUploadRequest(t, ts.URL+"/", "upload.json", newContent)

	done := make(chan struct{})
	var wg sync.WaitGroup
	errCh := make(chan error, 64)
	var firstDownload sync.Once
	firstDownloadDone := make(chan struct{})

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				case <-time.After(5 * time.Millisecond):
				}

				resp, err := client.Get(targetURL)
				if err != nil {
					errCh <- err
					return
				}
				var body bytes.Buffer
				_, readErr := body.ReadFrom(resp.Body)
				resp.Body.Close()
				if readErr != nil {
					errCh <- readErr
					return
				}
				got := body.Bytes()
				if !bytes.Equal(got, oldContent) && !bytes.Equal(got, newContent) {
					errCh <- fmt.Errorf("download saw a partial file: %d bytes", len(got))
					return
				}
				firstDownload.Do(func() { close(firstDownloadDone) })
			}
		}()
	}

	// Ensure downloads are already in flight when the upload starts.
	select {
	case <-firstDownloadDone:
	case <-time.After(10 * time.Second):
		close(done)
		wg.Wait()
		t.Fatal("downloads did not start")
	}

	uploadResult := make(chan error, 1)
	go func() {
		resp, err := client.Do(uploadReq)
		if err != nil {
			uploadResult <- err
			return
		}
		defer resp.Body.Close()
		var body bytes.Buffer
		body.ReadFrom(resp.Body)
		if resp.StatusCode != http.StatusOK {
			uploadResult <- fmt.Errorf("upload status = %d, body = %s", resp.StatusCode, body.String())
			return
		}
		uploadResult <- nil
	}()

	err := <-uploadResult
	close(done)
	wg.Wait()
	close(errCh)

	for downloadErr := range errCh {
		t.Error(downloadErr)
	}
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, newContent) {
		t.Fatalf("uploaded file content is not the new content: %d bytes", len(got))
	}
	if leftovers := listUploadTempFiles(t, root); len(leftovers) != 0 {
		t.Fatalf("upload left temporary files behind: %v", leftovers)
	}
}

func TestUploadFailureLeavesNoTempFile(t *testing.T) {
	root := t.TempDir()
	server := NewHTTPStaticServer(root, true)
	server.Upload = true
	ts := httptest.NewServer(server)
	defer ts.Close()

	// A directory at the destination path makes the final rename fail.
	dstPath := filepath.Join(root, "upload.json")
	if err := os.Mkdir(dstPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dstPath, "keep"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(newUploadRequest(t, ts.URL+"/", "upload.json", []byte("content")))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("upload status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
	if leftovers := listUploadTempFiles(t, root); len(leftovers) != 0 {
		t.Fatalf("failed upload left temporary files behind: %v", leftovers)
	}
	if data, err := os.ReadFile(filepath.Join(dstPath, "keep")); err != nil || string(data) != "keep" {
		t.Fatalf("failed upload modified the destination: %v, %q", err, data)
	}
}

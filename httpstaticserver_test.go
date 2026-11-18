package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

// TestFileListMediaThumbnails guards the minimal thumbnail feature from issue
// #133: images and videos get a thumbnail in the name cell, everything else
// keeps its icon, and no thumbnail autoplays.
func TestFileListMediaThumbnails(t *testing.T) {
	script, err := os.ReadFile("assets/js/index.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(script)
	for _, want := range []string{
		"var imageExtensions =",
		"var videoExtensions =",
		"isImage: function",
		"isVideo: function",
		"genThumbURL: function",
		"url + '#t=0.1'",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("assets/js/index.js must contain %q for media thumbnails", want)
		}
	}
	// The extension lists must live in one place: neither the icon switch nor
	// the click handler may keep their own copy.
	if strings.Contains(js, "var fileExtension") {
		t.Fatal("clickFileOrDir must use isVideo instead of its own extension list")
	}
	if got := strings.Count(js, `"fa-file-picture-o"`); got != 1 {
		t.Fatalf("image icon must be returned from one place, found %d occurrences", got)
	}
	if !strings.Contains(js, "if (this.isVideo(f)) {") {
		t.Fatal("clickFileOrDir must route videos through isVideo")
	}

	page, err := os.ReadFile("assets/index.html")
	if err != nil {
		t.Fatal(err)
	}
	template := string(page)
	for _, want := range []string{
		`class="file-thumb"`,
		`v-if="isImage(f)"`,
		`v-if="isVideo(f)"`,
		`v-bind:src="genThumbURL(f)"`,
		`loading="lazy"`,
		`preload="metadata"`,
		`muted`,
		`v-if="!isImage(f) && !isVideo(f)"`,
	} {
		if !strings.Contains(template, want) {
			t.Fatalf("assets/index.html must contain %q for media thumbnails", want)
		}
	}
	if got := strings.Count(template, `class="file-thumb"`); got != 2 {
		t.Fatalf("index.html must render one image and one video thumbnail, found %d thumbnails", got)
	}
	if strings.Contains(template, "autoplay") {
		t.Fatal("list thumbnails must not autoplay")
	}

	style, err := os.ReadFile("assets/css/style.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(style)
	for _, want := range []string{
		".file-thumb",
		"object-fit: cover",
		"width: 48px",
		"width: 36px",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("assets/css/style.css must contain %q for media thumbnails", want)
		}
	}
}

func TestFileListPaginationWiring(t *testing.T) {
	script, err := os.ReadFile("assets/js/index.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"page: 1",
		"pageSize: 100",
		"pageSizeOptions: [50, 100, 200]",
		"totalPages: function",
		"pagedFiles: function",
		"goToPage: function",
		"clampPage: function",
	} {
		if !strings.Contains(string(script), want) {
			t.Fatalf("assets/js/index.js must contain %q for file list pagination", want)
		}
	}

	page, err := os.ReadFile("assets/index.html")
	if err != nil {
		t.Fatal(err)
	}
	template := string(page)
	for _, want := range []string{
		`v-for="f in pagedFiles"`,
		`class="list-pagination text-center" v-if="totalPages > 1"`,
		`v-model="pageSize"`,
		`goToPage(page - 1)`,
		`goToPage(page + 1)`,
	} {
		if !strings.Contains(template, want) {
			t.Fatalf("assets/index.html must contain %q for file list pagination", want)
		}
	}
	if strings.Contains(template, `v-for="f in computedFiles"`) {
		t.Fatal("file list must render the paginated slice, not the full filtered list")
	}
	// The pagination controls belong below the file list, not in the toolbar above it.
	tableEnd := strings.Index(template, "</table>")
	paginationAt := strings.Index(template, `list-pagination`)
	if tableEnd == -1 || paginationAt == -1 || paginationAt < tableEnd {
		t.Fatal("file list pagination must be rendered below the table")
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

// TestUploadLargeFileDoesNotBufferForm reproduces issue #200: uploading a large
// file must not go through ParseMultipartForm, which keeps up to 32MB of the
// body in memory per request. Allocations are measured on the heap
// (TotalAlloc) rather than RSS, which never shrinks on its own.
func TestUploadLargeFileDoesNotBufferForm(t *testing.T) {
	root := t.TempDir()
	server := NewHTTPStaticServer(root, true)
	server.Upload = true
	ts := httptest.NewServer(server)
	defer ts.Close()

	const (
		fileSize  = 40 << 20 // above the 32MB defaultMaxMemory budget
		memBudget = 16 << 20
		chunkSize = 64 << 10
	)
	chunk := bytes.Repeat([]byte("0123456789abcdef"), chunkSize/16)
	sentHash := sha256.New()

	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	writeErr := make(chan error, 1)
	go func() {
		part, err := writer.CreateFormFile("file", "large.bin")
		if err != nil {
			pipeWriter.CloseWithError(err)
			writeErr <- err
			return
		}
		for written := 0; written < fileSize; written += chunkSize {
			if _, err := part.Write(chunk); err != nil {
				pipeWriter.CloseWithError(err)
				writeErr <- err
				return
			}
			sentHash.Write(chunk)
		}
		if err := writer.Close(); err != nil {
			pipeWriter.CloseWithError(err)
			writeErr <- err
			return
		}
		writeErr <- nil
		pipeWriter.Close()
	}()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", pipeReader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, readErr := io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	runtime.ReadMemStats(&after)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if err := <-writeErr; err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > memBudget {
		t.Fatalf("uploading %d bytes allocated %d bytes, want < %d (the form must be streamed, not buffered)",
			fileSize, allocated, memBudget)
	}

	dstPath := filepath.Join(root, "large.bin")
	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != fileSize {
		t.Fatalf("uploaded file size = %d, want %d", info.Size(), fileSize)
	}
	file, err := os.Open(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gotHash := sha256.New()
	if _, err := io.Copy(gotHash, file); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotHash.Sum(nil), sentHash.Sum(nil)) {
		t.Fatal("uploaded file content does not match what was sent")
	}
	if leftovers := listUploadTempFiles(t, root); len(leftovers) != 0 {
		t.Fatalf("upload left temporary files behind: %v", leftovers)
	}
}

type uploadField struct {
	name  string
	value string
}

func newUploadRequestWithFields(t *testing.T, url, filename string, content []byte, before, after []uploadField) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, field := range before {
		if err := writer.WriteField(field.name, field.value); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(content); err != nil {
		t.Fatal(err)
	}
	for _, field := range after {
		if err := writer.WriteField(field.name, field.value); err != nil {
			t.Fatal(err)
		}
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

// TestUploadFieldOrder covers #200's streaming handler: the file part is
// written before the rest of the body is known, so filename/token fields must
// work both before and after it, and a rejected upload must not leave anything
// on disk.
func TestUploadFieldOrder(t *testing.T) {
	root := t.TempDir()
	conf := "upload: false\nusers:\n" +
		"  - email: user@example.com\n" +
		"    upload: true\n" +
		"    token: secret\n"
	if err := os.WriteFile(filepath.Join(root, YAMLCONF), []byte(conf), 0644); err != nil {
		t.Fatal(err)
	}
	server := NewHTTPStaticServer(root, true)
	ts := httptest.NewServer(server)
	defer ts.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	tests := []struct {
		name    string
		target  string
		before  []uploadField
		after   []uploadField
		status  int
		content string
	}{
		{
			name:   "filename before token after",
			target: "before.txt",
			before: []uploadField{{"filename", "before.txt"}},
			after:  []uploadField{{"token", "secret"}},
			status: http.StatusOK,
		},
		{
			name:   "token before filename after",
			target: "after.txt",
			before: []uploadField{{"token", "secret"}},
			after:  []uploadField{{"filename", "after.txt"}},
			status: http.StatusOK,
		},
		{
			name:   "wrong token",
			target: "forbidden.txt",
			before: []uploadField{{"filename", "forbidden.txt"}},
			after:  []uploadField{{"token", "wrong"}},
			status: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.Do(newUploadRequestWithFields(t, ts.URL+"/", "ignored.txt", []byte(tc.name), tc.before, tc.after))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.status {
				t.Fatalf("upload status = %d, want %d", resp.StatusCode, tc.status)
			}

			dstPath := filepath.Join(root, tc.target)
			if tc.status != http.StatusOK {
				if _, err := os.Stat(dstPath); !os.IsNotExist(err) {
					t.Fatalf("rejected upload created %s", dstPath)
				}
				if leftovers := listUploadTempFiles(t, root); len(leftovers) != 0 {
					t.Fatalf("rejected upload left temporary files behind: %v", leftovers)
				}
				return
			}
			data, err := os.ReadFile(dstPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tc.name {
				t.Fatalf("uploaded content = %q, want %q", data, tc.name)
			}
		})
	}
}

// TestUploadMkdirWithoutBody keeps the "POST without a file only creates the
// directory" contract that the dashboard's new-directory button relies on.
func TestUploadMkdirWithoutBody(t *testing.T) {
	root := t.TempDir()
	server := NewHTTPStaticServer(root, true)
	server.Upload = true
	ts := httptest.NewServer(server)
	defer ts.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(ts.URL+"/newdir", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mkdir status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if !isDir(filepath.Join(root, "newdir")) {
		t.Fatal("mkdir request did not create the directory")
	}

	server.Upload = false
	resp2, err := client.Post(ts.URL+"/denied", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusForbidden {
		t.Fatalf("mkdir without upload permission status = %d, want %d", resp2.StatusCode, http.StatusForbidden)
	}
	if _, err := os.Stat(filepath.Join(root, "denied")); !os.IsNotExist(err) {
		t.Fatal("forbidden mkdir request created the directory")
	}
}

// TestUploadForbiddenCleansUp guards the streaming handler's only downside: an
// unauthorized upload is written to disk before its token is known, so nothing
// it created may survive the 403.
func TestUploadForbiddenCleansUp(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, YAMLCONF), []byte("upload: false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	server := NewHTTPStaticServer(root, true)
	ts := httptest.NewServer(server)
	defer ts.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	req := newUploadRequestWithFields(t, ts.URL+"/newdir", "upload.txt", bytes.Repeat([]byte("x"), 1<<20),
		nil, []uploadField{{"token", "wrong"}})
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("upload status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
	if _, err := os.Stat(filepath.Join(root, "newdir")); !os.IsNotExist(err) {
		t.Fatal("forbidden upload left the directory it created behind")
	}
	if leftovers := listUploadTempFiles(t, root); len(leftovers) != 0 {
		t.Fatalf("forbidden upload left temporary files behind: %v", leftovers)
	}
}

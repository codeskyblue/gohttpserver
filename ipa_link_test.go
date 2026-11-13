package main

import (
	"archive/zip"
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const testIPAInfoPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleIdentifier</key>
	<string>com.example.test</string>
	<key>CFBundleVersion</key>
	<string>1.0</string>
	<key>CFBundleName</key>
	<string>Test</string>
</dict>
</plist>
`

func writeTestIPA(t *testing.T, dir, name string) {
	t.Helper()

	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create("Payload/Test.app/Info.plist")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(testIPAInfoPlist)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

// responseText returns the response body with the JavaScript escaping applied by
// html/template to URLs undone.
func responseText(recorder *httptest.ResponseRecorder) string {
	return strings.ReplaceAll(recorder.Body.String(), `\/`, "/")
}

func newTestServer(t *testing.T) (*HTTPStaticServer, string) {
	t.Helper()

	root, err := os.MkdirTemp("", "gohttpserver-ipa-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })

	return NewHTTPStaticServer(root, true), root
}

func TestHIpaLinkDoesNotFetchAttackerHost(t *testing.T) {
	server, root := newTestServer(t)
	writeTestIPA(t, root, "Test.ipa")

	var (
		mu       sync.Mutex
		postBody string
		posted   bool
	)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		mu.Lock()
		posted = true
		postBody = string(body)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"key":"abc123"}`))
	}))
	defer proxy.Close()
	server.PlistProxy = proxy.URL

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/-/ipa/link/Test.ipa", nil)
	request.Host = "169.254.169.254"
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	mu.Lock()
	defer mu.Unlock()
	if !posted {
		t.Fatal("plist proxy was not called")
	}
	if !strings.Contains(postBody, "software-package") || !strings.Contains(postBody, "com.example.test") {
		t.Fatalf("proxy did not receive the generated plist: %s", postBody)
	}

	body := responseText(recorder)
	wantLink := proxy.URL + "/abc123"
	if !strings.Contains(body, wantLink) {
		t.Fatalf("response does not contain %q: %s", wantLink, body)
	}

	for _, leak := range []string{"dial tcp", "no such host", "connection refused", "169.254.169.254"} {
		if strings.Contains(body, leak) {
			t.Fatalf("response leaks %q: %s", leak, body)
		}
	}
}

func TestHIpaLinkHidesUpstreamErrors(t *testing.T) {
	server, root := newTestServer(t)
	writeTestIPA(t, root, "Test.ipa")

	// Reserve a port and close it so the proxy connection is refused.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	unreachable := "http://" + listener.Addr().String()
	listener.Close()
	server.PlistProxy = unreachable

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/-/ipa/link/Test.ipa", nil)
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != "plist generation failed" {
		t.Fatalf("body = %q, want generic error", got)
	}

	// A missing IPA must not disclose filesystem paths either.
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/-/ipa/link/Missing.ipa", nil)
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != "plist generation failed" {
		t.Fatalf("body = %q, want generic error", got)
	}
	if strings.Contains(recorder.Body.String(), root) {
		t.Fatalf("response leaks server path: %s", recorder.Body.String())
	}
}

func TestHIpaLinkUsesTLSScheme(t *testing.T) {
	server, root := newTestServer(t)
	writeTestIPA(t, root, "Test.ipa")

	// No proxy: over TLS the handler must return a direct plist link instead.
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/-/ipa/link/Test.ipa", nil)
	request.TLS = &tls.ConnectionState{}
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	body := responseText(recorder)
	wantLink := "https://example.com/-/ipa/plist/Test.ipa"
	if !strings.Contains(body, wantLink) {
		t.Fatalf("response does not contain %q: %s", wantLink, body)
	}
	if strings.Contains(body, "http://example.com/-/ipa/plist/") {
		t.Fatalf("TLS request produced an http:// plist link: %s", body)
	}
}

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
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
		`v-if="totalPages > 1"`,
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
}

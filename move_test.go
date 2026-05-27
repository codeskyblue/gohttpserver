package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestServer(t *testing.T) (*HTTPStaticServer, string, func()) {
	rootDir, err := ioutil.TempDir("", "ghs_test_")
	if err != nil {
		t.Fatal(err)
	}

	// Create some files and directories
	os.MkdirAll(filepath.Join(rootDir, "subdir"), 0755)
	ioutil.WriteFile(filepath.Join(rootDir, "file1.txt"), []byte("content1"), 0644)
	ioutil.WriteFile(filepath.Join(rootDir, "subdir", "file2.txt"), []byte("content2"), 0644)

	// Create config for permissions (allow delete and upload)
	configContent := `
upload: true
delete: true
`
	ioutil.WriteFile(filepath.Join(rootDir, ".ghs.yml"), []byte(configContent), 0644)

	s := NewHTTPStaticServer(rootDir, true)
	s.Upload = true
	s.Delete = true

	cleanup := func() {
		os.RemoveAll(rootDir)
	}

	return s, rootDir, cleanup
}

func TestMoveFile(t *testing.T) {
	s, root, cleanup := setupTestServer(t)
	defer cleanup()

	// Move file1.txt to file1_moved.txt
	data := url.Values{}
	data.Set("src", "file1.txt")
	data.Set("dst", "file1_moved.txt")

	req, _ := http.NewRequest("POST", "/-/move", nil)
	req.URL.RawQuery = data.Encode()
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify file moved
	_, err := os.Stat(filepath.Join(root, "file1.txt"))
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(root, "file1_moved.txt"))
	assert.NoError(t, err)
}

func TestMoveDirectory(t *testing.T) {
	s, root, cleanup := setupTestServer(t)
	defer cleanup()

	// Move subdir to subdir_moved
	data := url.Values{}
	data.Set("src", "subdir")
	data.Set("dst", "subdir_moved")

	req, _ := http.NewRequest("POST", "/-/move", nil)
	req.URL.RawQuery = data.Encode()
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify dir moved
	_, err := os.Stat(filepath.Join(root, "subdir"))
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(root, "subdir_moved"))
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(root, "subdir_moved", "file2.txt"))
	assert.NoError(t, err)
}

func TestMoveToNewDirectory(t *testing.T) {
	s, root, cleanup := setupTestServer(t)
	defer cleanup()

	// Move file1.txt to newdir/file1.txt (newdir does not exist)
	data := url.Values{}
	data.Set("src", "file1.txt")
	data.Set("dst", "newdir/file1.txt")

	req, _ := http.NewRequest("POST", "/-/move", nil)
	req.URL.RawQuery = data.Encode()
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify file moved
	_, err := os.Stat(filepath.Join(root, "file1.txt"))
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(root, "newdir", "file1.txt"))
	assert.NoError(t, err)
}

func TestMoveConflict(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Try to move file1.txt to subdir/file2.txt (which exists)
	data := url.Values{}
	data.Set("src", "file1.txt")
	data.Set("dst", "subdir/file2.txt")

	req, _ := http.NewRequest("POST", "/-/move", nil)
	req.URL.RawQuery = data.Encode()
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestMoveOverwrite(t *testing.T) {
	s, root, cleanup := setupTestServer(t)
	defer cleanup()

	// file1.txt exists, subdir/file2.txt exists
	// Move file1.txt to subdir/file2.txt with overwrite=true
	data := url.Values{}
	data.Set("src", "file1.txt")
	data.Set("dst", "subdir/file2.txt")
	data.Set("overwrite", "true")

	req, _ := http.NewRequest("POST", "/-/move", nil)
	req.URL.RawQuery = data.Encode()
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify file moved and overwritten
	_, err := os.Stat(filepath.Join(root, "file1.txt"))
	assert.True(t, os.IsNotExist(err))

	content, err := ioutil.ReadFile(filepath.Join(root, "subdir", "file2.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "content1", string(content))
}

func TestMoveSourceNotFound(t *testing.T) {
	s, _, cleanup := setupTestServer(t)
	defer cleanup()

	data := url.Values{}
	data.Set("src", "nonexistent.txt")
	data.Set("dst", "somewhere.txt")

	req, _ := http.NewRequest("POST", "/-/move", nil)
	req.URL.RawQuery = data.Encode()
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMoveForbidden(t *testing.T) {
	s, root, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a read-only config
	configContent := `
upload: false
delete: false
`
	ioutil.WriteFile(filepath.Join(root, ".ghs.yml"), []byte(configContent), 0644)

	data := url.Values{}
	data.Set("src", "file1.txt")
	data.Set("dst", "file1_moved.txt")

	req, _ := http.NewRequest("POST", "/-/move", nil)
	req.URL.RawQuery = data.Encode()
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	// Since we check delete permission first
	assert.Equal(t, http.StatusForbidden, w.Code)
}

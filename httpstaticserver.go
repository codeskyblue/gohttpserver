package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

type HTTPStaticServer struct {
	Root string
	m    *mux.Router
}

func NewHTTPStaticServer(root string) *HTTPStaticServer {
	if root == "" {
		root = "."
	}
	m := mux.NewRouter()
	s := &HTTPStaticServer{
		Root: root,
		m:    m,
	}
	m.HandleFunc("/-/res/{path:.*}", s.hAssets)
	m.HandleFunc("/-/raw/{path:.*}", s.hFileOrDirectory)
	m.HandleFunc("/-/json/{path:.*}", s.hJSONList)
	m.HandleFunc("/{path:.*}", s.hIndex).Methods("GET")
	return s
}

func (s *HTTPStaticServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.ServeHTTP(w, r)
}

func (s *HTTPStaticServer) hIndex(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	relPath := filepath.Join(s.Root, path)
	finfo, err := os.Stat(relPath)
	if err == nil && finfo.IsDir() {
		indexPath := filepath.Join("./res", "index.tmpl.html")
		t := template.New("index").Delims("[[", "]]")
		tmpl := template.Must(t.ParseFiles(indexPath))
		tmpl.ExecuteTemplate(w, "index.tmpl.html", nil)
	} else {
		http.ServeFile(w, r, relPath)
	}
}

func (s *HTTPStaticServer) hAssets(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	// w.Header().Set("Content-Type", "text/plain") // -_-! not working in chrome
	http.ServeFile(w, r, filepath.Join("./res", path))
}

func (s *HTTPStaticServer) hFileOrDirectory(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	log.Println("Path:", s.Root, path)
	http.ServeFile(w, r, filepath.Join(s.Root, path))
}

type ListResponse struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
	Size string `json:"size"`
}

func (s *HTTPStaticServer) hJSONList(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	lrs := make([]ListResponse, 0)
	fd, err := os.Open(filepath.Join(s.Root, path))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer fd.Close()

	files, err := fd.Readdir(-1)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	for _, file := range files {
		lr := ListResponse{
			Name: file.Name(),
			Path: filepath.Join(path, file.Name()), // lstrip "/"
		}
		if file.IsDir() {
			lr.Type = "dir"
			lr.Size = "-"
		} else {
			lr.Type = "file"
			lr.Size = formatSize(file)
		}
		lrs = append(lrs, lr)
	}

	data, _ := json.Marshal(lrs)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

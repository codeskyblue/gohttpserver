package main

import (
	"html/template"
	"log"
	"net/http"
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
	m.HandleFunc("/", s.hIndex)
	m.HandleFunc("/assets/{path:.*}", s.hAssets)
	m.HandleFunc("/raw/{path:.*}", s.hFileOrDirectory)
	return s
}

func (s *HTTPStaticServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.ServeHTTP(w, r)
}

func (s *HTTPStaticServer) hIndex(w http.ResponseWriter, r *http.Request) {
	indexPath := filepath.Join("./assets", "index.html")
	tmpl := template.Must(template.New("index").ParseFiles(indexPath))
	tmpl.ExecuteTemplate(w, "index.html", nil)
}

func (s *HTTPStaticServer) hAssets(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	w.Header().Set("Content-Type", "text/plain") // -_-! not working in chrome
	http.ServeFile(w, r, filepath.Join("./assets", path))
}

func (s *HTTPStaticServer) hFileOrDirectory(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	log.Println("Path:", s.Root, path)
	http.ServeFile(w, r, filepath.Join(s.Root, path))
}

package main

import (
	"encoding/json"
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
	m.HandleFunc("/-/res/{path:.*}", s.hAssets)
	m.HandleFunc("/-/raw/{path:.*}", s.hFileOrDirectory)
	m.HandleFunc("/-/json/{path:.*}", s.hJSONList)
	return s
}

func (s *HTTPStaticServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.ServeHTTP(w, r)
}

func (s *HTTPStaticServer) hIndex(w http.ResponseWriter, r *http.Request) {
	indexPath := filepath.Join("./res", "index.tmpl")
	t := template.New("index").Delims("[[", "]]")
	tmpl := template.Must(t.ParseFiles(indexPath))
	tmpl.ExecuteTemplate(w, "index.tmpl", nil)
}

func (s *HTTPStaticServer) hAssets(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	w.Header().Set("Content-Type", "text/plain") // -_-! not working in chrome
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
}

func (s *HTTPStaticServer) hJSONList(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	lr := ListResponse{
		Name: "Hello",
		Path: path,
	}
	data, _ := json.Marshal(lr)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

type HTTPStaticServer struct {
	Root string
	m    *mux.Router
}

func NewHTTPStaticServer(root string) *HTTPStaticServer {
	m := mux.NewRouter()
	s := &HTTPStaticServer{
		Root: root,
		m:    m,
	}
	m.HandleFunc("/", s.hIndex)
	return s
}

func (s *HTTPStaticServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.ServeHTTP(w, r)
}

func (s *HTTPStaticServer) hIndex(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello world!"))
}

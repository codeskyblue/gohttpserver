package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

type HTTPStaticServer struct {
	Root  string
	Theme string

	m *mux.Router
}

func NewHTTPStaticServer(root string, theme string) *HTTPStaticServer {
	if root == "" {
		root = "."
	}
	m := mux.NewRouter()
	s := &HTTPStaticServer{
		Root:  root,
		Theme: theme, // TODO: need to parse from command line
		m:     m,
	}
	m.HandleFunc("/-/status", s.hStatus)
	m.HandleFunc("/-/raw/{path:.*}", s.hFileOrDirectory)
	m.HandleFunc("/-/zip/{path:.*}", s.hZip)
	m.HandleFunc("/-/json/{path:.*}", s.hJSONList)
	// routers for Apple *.ipa
	m.HandleFunc("/-/ipa/icon/{path:.*}", s.hIpaIcon)
	m.HandleFunc("/-/ipa/plist/{path:.*}", s.hPlist)
	// TODO: /ipa/link, /ipa/info

	m.HandleFunc("/{path:.*}", s.hIndex).Methods("GET")
	return s
}

func (s *HTTPStaticServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.ServeHTTP(w, r)
}

func (s *HTTPStaticServer) hStatus(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello world"))
}

func (s *HTTPStaticServer) hIndex(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	relPath := filepath.Join(s.Root, path)
	finfo, err := os.Stat(relPath)
	if err == nil && finfo.IsDir() {
		tmpl.Execute(w, s)
	} else {
		http.ServeFile(w, r, relPath)
	}
}

func (s *HTTPStaticServer) hZip(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	CompressToZip(w, filepath.Join(s.Root, path))
}

func (s *HTTPStaticServer) hIpaIcon(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	relPath := filepath.Join(s.Root, path)
	data, err := parseIpaIcon(relPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound) // If parse icon error, 404 maybe the best way.
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(data)
}

func (s *HTTPStaticServer) hPlist(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	// rename *.plist to *.ipa
	if filepath.Ext(path) == ".plist" {
		path = path[0:len(path)-6] + ".ipa"
	}

	relPath := filepath.Join(s.Root, path)
	plinfo, err := parseIPA(relPath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	ipaURL := url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   path,
	}
	imgURL := url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   filepath.Join("/-/ipa/icon", path),
	}
	// TODO: image ignore here.
	data, err := generateDownloadPlist(ipaURL.String(), imgURL.String(), plinfo)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/xml")
	w.Write(data)
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
			fileName := deepPath(filepath.Join(s.Root, path), file.Name())
			lr.Name = fileName
			lr.Path = filepath.Join(path, fileName)
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

func deepPath(basedir, name string) string {
	isDir := true
	// loop max 5, incase of for loop not finished
	maxDepth := 5
	for depth := 0; depth <= maxDepth && isDir; depth += 1 {
		finfos, err := ioutil.ReadDir(filepath.Join(basedir, name))
		if err != nil || len(finfos) != 1 {
			break
		}
		if finfos[0].IsDir() {
			name = filepath.Join(name, finfos[0].Name())
		} else {
			break
		}
	}
	return name
}

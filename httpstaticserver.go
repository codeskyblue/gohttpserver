package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type IndexFileItem struct {
	Path string
	Info os.FileInfo
}

type HTTPStaticServer struct {
	Root       string
	Theme      string
	Upload     bool
	Title      string
	PlistProxy string

	indexes []IndexFileItem
	m       *mux.Router
}

func NewHTTPStaticServer(root string) *HTTPStaticServer {
	if root == "" {
		root = "."
	}
	root = filepath.ToSlash(root)
	if !strings.HasSuffix(root, "/") {
		root = root + "/"
	}
	m := mux.NewRouter()
	s := &HTTPStaticServer{
		Root:  root,
		Theme: "black",
		m:     m,
	}

	go func() {
		for {
			log.Println("making fs index ...")
			s.makeIndex()
			log.Println("indexing finished, next index after 10 minutes")
			//time.Sleep(time.Second * 1)
			time.Sleep(time.Minute * 10)
		}
	}()

	m.HandleFunc("/-/status", s.hStatus)
	m.HandleFunc("/-/raw/{path:.*}", s.hFileOrDirectory)
	m.HandleFunc("/-/zip/{path:.*}", s.hZip)
	m.HandleFunc("/-/unzip/{zip_path:.*}/-/{path:.*}", s.hUnzip)
	m.HandleFunc("/-/json/{path:.*}", s.hJSONList)
	// routers for Apple *.ipa
	m.HandleFunc("/-/ipa/plist/{path:.*}", s.hPlist)
	m.HandleFunc("/-/ipa/link/{path:.*}", s.hIpaLink)
	// TODO: /ipa/info

	m.HandleFunc("/{path:.*}", s.hIndex).Methods("GET")
	return s
}

func (s *HTTPStaticServer) EnableUpload() {
	s.Upload = true
	s.m.HandleFunc("/{path:.*}", s.hUpload).Methods("POST")
}

func (s *HTTPStaticServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.ServeHTTP(w, r)
}

func (s *HTTPStaticServer) hIndex(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	relPath := filepath.Join(s.Root, path)

	finfo, err := os.Stat(relPath)
	if err == nil && finfo.IsDir() {
		tmpl.ExecuteTemplate(w, "index", s)
	} else {
		http.ServeFile(w, r, relPath)
	}
}

func (s *HTTPStaticServer) hStatus(w http.ResponseWriter, r *http.Request) {
	data, _ := json.MarshalIndent(s, "", "    ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *HTTPStaticServer) hUpload(w http.ResponseWriter, req *http.Request) {
	err := req.ParseMultipartForm(1 << 30) // max memory 1G
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(req.MultipartForm.File["file"]) == 0 {
		http.Error(w, "Need multipart file", http.StatusInternalServerError)
		return
	}

	path := mux.Vars(req)["path"]
	dirpath := filepath.Join(s.Root, path)

	for _, mfile := range req.MultipartForm.File["file"] {
		file, err := mfile.Open()
		defer file.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		dst, err := os.Create(filepath.Join(dirpath, mfile.Filename)) // BUG(ssx): There is a leak here
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, file); err != nil {
			log.Println("Handle upload file:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.Write([]byte("Upload success"))
}

func (s *HTTPStaticServer) hZip(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	CompressToZip(w, filepath.Join(s.Root, path))
}

func (s *HTTPStaticServer) hUnzip(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	zipPath, path := vars["zip_path"], vars["path"]
	ctype := mime.TypeByExtension(filepath.Ext(path))
	if ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	err := ExtractFromZip(filepath.Join(s.Root, zipPath), path, w)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func genURLStr(r *http.Request, path string) *url.URL {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return &url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   path,
	}
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
	baseURL := &url.URL{
		Scheme: scheme,
		Host:   r.Host,
	}
	data, err := generateDownloadPlist(baseURL, path, plinfo)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/xml")
	w.Write(data)
}

func (s *HTTPStaticServer) hIpaLink(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	plistUrl := genURLStr(r, "/-/ipa/plist/"+path).String()
	if s.PlistProxy != "" {
		plistUrl = strings.TrimSuffix(s.PlistProxy, "/") + "/" + r.Host + "/-/ipa/plist/" + path
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl.ExecuteTemplate(w, "ipa-install", map[string]string{
		"Name":      filepath.Base(path),
		"PlistLink": plistUrl,
	})
	// w.Write([]byte(fmt.Sprintf(
	// 	`<a href='itms-services://?action=download-manifest&url=%s'>Click this link to install</a>`,
	// 	plistUrl)))
}

func (s *HTTPStaticServer) hFileOrDirectory(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	http.ServeFile(w, r, filepath.Join(s.Root, path))
}

type ListResponse struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
	Size string `json:"size"`
}

func (s *HTTPStaticServer) hJSONList(w http.ResponseWriter, r *http.Request) {
	requestPath := mux.Vars(r)["path"]
	localPath := filepath.Join(s.Root, requestPath)
	search := r.FormValue("search")

	// path string -> info os.FileInfo
	fileInfoMap := make(map[string]os.FileInfo, 0)

	if search != "" {
		results := s.findIndex(search)
		if len(results) > 50 { // max 50
			results = results[:50]
		}
		for _, item := range results {
			if filepath.HasPrefix(item.Path, requestPath) {
				fileInfoMap[item.Path] = item.Info
			}
		}
	} else {
		fd, err := os.Open(localPath)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer fd.Close()

		infos, err := fd.Readdir(-1)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		for _, info := range infos {
			fileInfoMap[filepath.Join(requestPath, info.Name())] = info
		}
	}

	lrs := make([]ListResponse, 0)
	for path, info := range fileInfoMap {
		lr := ListResponse{
			Name: info.Name(),
			Path: path,
		}
		if search != "" {
			name, err := filepath.Rel(requestPath, path)
			if err != nil {
				log.Println(requestPath, path, err)
			}
			lr.Name = filepath.ToSlash(name) // fix for windows
		}
		if info.IsDir() {
			name := deepPath(localPath, info.Name())
			lr.Name = name
			lr.Path = filepath.Join(filepath.Dir(path), name)
			lr.Type = "dir"
			lr.Size = "-"
		} else {
			lr.Type = "file"
			lr.Size = formatSize(info)
		}
		lrs = append(lrs, lr)
	}

	data, _ := json.Marshal(lrs)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *HTTPStaticServer) makeIndex() error {
	var indexes = make([]IndexFileItem, 0)
	var err = filepath.Walk(s.Root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.IsAbs(path) {
			path, _ = filepath.Rel(s.Root, path)
		}
		path = filepath.ToSlash(path)
		indexes = append(indexes, IndexFileItem{path, info})
		return nil
	})
	s.indexes = indexes
	return err
}

func (s *HTTPStaticServer) findIndex(text string) []IndexFileItem {
	ret := make([]IndexFileItem, 0)
	for _, item := range s.indexes {
		ok := true
		// search algorithm, space for AND
		for _, keyword := range strings.Fields(text) {
			if !strings.Contains(strings.ToLower(item.Path), strings.ToLower(keyword)) {
				ok = false
			}
		}
		if ok {
			ret = append(ret, item)
		}
	}
	return ret
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
			name = filepath.ToSlash(filepath.Join(name, finfos[0].Name()))
		} else {
			break
		}
	}
	return name
}

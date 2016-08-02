package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/gorilla/mux"
)

type IndexFileItem struct {
	Path string
	Info os.FileInfo
}

type HTTPStaticServer struct {
	Root            string
	Theme           string
	Upload          bool
	Title           string
	PlistProxy      string
	GoogleTrackerId string

	indexes []IndexFileItem
	m       *mux.Router
}

func NewHTTPStaticServer(root string) *HTTPStaticServer {
	if root == "" {
		root = "./"
	}
	root = filepath.ToSlash(root)
	if !strings.HasSuffix(root, "/") {
		root = root + "/"
	}
	log.Printf("root path: %s\n", root)
	m := mux.NewRouter()
	s := &HTTPStaticServer{
		Root:  root,
		Theme: "black",
		m:     m,
	}

	go func() {
		time.Sleep(1 * time.Second)
		for {
			startTime := time.Now()
			log.Println("Started making search index")
			s.makeIndex()
			log.Printf("Completed search index in %v", time.Since(startTime))
			//time.Sleep(time.Second * 1)
			time.Sleep(time.Minute * 10)
		}
	}()

	m.HandleFunc("/-/status", s.hStatus)
	m.HandleFunc("/-/zip/{path:.*}", s.hZip)
	m.HandleFunc("/-/unzip/{zip_path:.*}/-/{path:.*}", s.hUnzip)
	m.HandleFunc("/-/json/{path:.*}", s.hJSONList)
	// routers for Apple *.ipa
	m.HandleFunc("/-/ipa/plist/{path:.*}", s.hPlist)
	m.HandleFunc("/-/ipa/link/{path:.*}", s.hIpaLink)
	// TODO: /ipa/info

	m.HandleFunc("/{path:.*}", s.hIndex).Methods("GET")
	m.HandleFunc("/{path:.*}", s.hUpload).Methods("POST")
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
		tmpl.ExecuteTemplate(w, "index", s)
	} else {
		if r.FormValue("download") == "true" {
			w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(filepath.Base(path)))
		}
		http.ServeFile(w, r, relPath)
	}
}

func (s *HTTPStaticServer) hStatus(w http.ResponseWriter, r *http.Request) {
	data, _ := json.MarshalIndent(s, "", "    ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *HTTPStaticServer) hUpload(w http.ResponseWriter, req *http.Request) {
	path := mux.Vars(req)["path"]

	// check auth
	auth := s.readAccessConf(path)
	if !auth.Upload {
		http.Error(w, "Upload forbidden", http.StatusForbidden)
		return
	}

	err := req.ParseMultipartForm(1 << 30) // max memory 1G
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(req.MultipartForm.File["file"]) == 0 {
		http.Error(w, "Need multipart file", http.StatusInternalServerError)
		return
	}

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
	if r.TLS == nil {
		// send plist to plistproxy and get a https link
		httpPlistLink := "http://" + r.Host + "/-/ipa/plist/" + path
		url, err := s.genPlistLink(httpPlistLink)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		plistUrl = url
		//plistUrl = strings.TrimSuffix(s.PlistProxy, "/") + "/" + r.Host + "/-/ipa/plist/" + path
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

func (s *HTTPStaticServer) genPlistLink(httpPlistLink string) (plistUrl string, err error) {
	// Maybe need a proxy, a little slowly now.
	pp := s.PlistProxy
	if pp == "" {
		pp = defaultPlistProxy
	}
	resp, err := http.Get(httpPlistLink)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	retData, err := http.Post(pp, "text/xml", bytes.NewBuffer(data))
	if err != nil {
		return
	}
	defer retData.Body.Close()

	jsonData, _ := ioutil.ReadAll(retData.Body)
	var ret map[string]string
	if err = json.Unmarshal(jsonData, &ret); err != nil {
		return
	}
	plistUrl = pp + "/" + ret["key"]
	return
}

func (s *HTTPStaticServer) hFileOrDirectory(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	http.ServeFile(w, r, filepath.Join(s.Root, path))
}

type ListResponse struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Type    string `json:"type"`
	Size    string `json:"size"`
	ModTime int64  `json:"mtime"`
}

type AccessConf struct {
	Upload bool `yaml:"upload" json:"upload"`
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
		infos, err := ioutil.ReadDir(localPath)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		for _, info := range infos {
			fileInfoMap[filepath.Join(requestPath, info.Name())] = info
		}
	}

	// turn file list -> json
	lrs := make([]ListResponse, 0)
	for path, info := range fileInfoMap {
		lr := ListResponse{
			Name:    info.Name(),
			Path:    path,
			ModTime: info.ModTime().UnixNano() / 1e6,
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

	data, _ := json.Marshal(map[string]interface{}{
		"files": lrs,
		"auth":  s.readAccessConf(requestPath),
	})
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
			needContains := true
			if strings.HasPrefix(keyword, "-") {
				needContains = false
				keyword = keyword[1:]
			}
			if keyword == "" {
				continue
			}
			ok = (needContains == strings.Contains(strings.ToLower(item.Path), strings.ToLower(keyword)))
			if !ok {
				break
			}
		}
		if ok {
			ret = append(ret, item)
		}
	}
	return ret
}

func (s *HTTPStaticServer) defaultAccessConf() AccessConf {
	return AccessConf{
		Upload: s.Upload,
	}
}

func (s *HTTPStaticServer) readAccessConf(requestPath string) (ac AccessConf) {
	ac = s.defaultAccessConf()
	cfgFile := filepath.Join(s.Root, requestPath, ".ghs.yml")
	data, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("Err read .ghs.yml: %v", err)
	}
	err = yaml.Unmarshal(data, &ac)
	if err != nil {
		log.Printf("Err format .ghs.yml: %v", err)
	}
	return
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

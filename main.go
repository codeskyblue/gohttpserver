package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/alecthomas/kingpin"
	"github.com/gorilla/mux"
)

type Configure struct {
	Addr     string
	Root     string
	HttpAuth string
	Cert     string
	Key      string
}

var gcfg = Configure{}

func FileHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "unknown")
}

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

func parseFlags() {
	kingpin.HelpFlag.Short('h')
	kingpin.Flag("addr", "listen address").Short('a').Default(":8000").StringVar(&gcfg.Addr)
	kingpin.Parse()
}

func main() {
	parseFlags()
	ss := NewHTTPStaticServer("/")

	log.Printf("Listening on addr: %s\n", strconv.Quote(gcfg.Addr))
	log.Fatal(http.ListenAndServe(gcfg.Addr, ss))
}

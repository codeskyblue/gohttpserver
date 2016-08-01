package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/alecthomas/kingpin"
	"github.com/goji/httpauth"
	"github.com/gorilla/handlers"
	accesslog "github.com/mash/go-accesslog"
)

type Configure struct {
	Addr       string
	Root       string
	HttpAuth   string
	Cert       string
	Key        string
	Cors       bool
	Theme      string
	XHeaders   bool
	Upload     bool
	PlistProxy *url.URL
	Title      string
}

type logger struct {
}

func (l logger) Log(record accesslog.LogRecord) {
	log.Printf("%s [code %d] %s", record.Method, record.Status, record.Uri)
}

var (
	defaultPlistProxy = "https://plistproxy.herokuapp.com/plist"
	gcfg              = Configure{}
	l                 = logger{}

	VERSION   = "unknown"
	BUILDTIME = "unknown time"
	GITCOMMIT = "unknown git commit"
	SITE      = "https://github.com/codeskyblue/gohttpserver"
)

func versionMessage() string {
	t := template.Must(template.New("version").Parse(`GoHTTPServer
  Version:        {{.Version}}
  Go version:     {{.GoVersion}}
  OS/Arch:        {{.OSArch}}
  Git commit:     {{.GitCommit}}
  Built:          {{.Built}}
  Site:           {{.Site}}`))
	buf := bytes.NewBuffer(nil)
	t.Execute(buf, map[string]interface{}{
		"Version":   VERSION,
		"GoVersion": runtime.Version(),
		"OSArch":    runtime.GOOS + "/" + runtime.GOARCH,
		"GitCommit": GITCOMMIT,
		"Built":     BUILDTIME,
		"Site":      SITE,
	})
	return buf.String()
}

func parseFlags() {
	kingpin.HelpFlag.Short('h')
	kingpin.Version(versionMessage())
	kingpin.Flag("root", "root directory").Short('r').Default("./").StringVar(&gcfg.Root)
	kingpin.Flag("addr", "listen address").Short('a').Default(":8000").StringVar(&gcfg.Addr)
	kingpin.Flag("cert", "tls cert.pem path").StringVar(&gcfg.Cert)
	kingpin.Flag("key", "tls key.pem path").StringVar(&gcfg.Key)
	kingpin.Flag("httpauth", "HTTP basic auth (ex: user:pass)").Default("").StringVar(&gcfg.HttpAuth)
	kingpin.Flag("theme", "web theme, one of <black|green>").Default("black").StringVar(&gcfg.Theme)
	kingpin.Flag("upload", "enable upload support").BoolVar(&gcfg.Upload)
	kingpin.Flag("xheaders", "used when behide nginx").BoolVar(&gcfg.XHeaders)
	kingpin.Flag("cors", "enable cross-site HTTP request").BoolVar(&gcfg.Cors)
	kingpin.Flag("plistproxy", "plist proxy when server is not https").Default(defaultPlistProxy).Short('p').URLVar(&gcfg.PlistProxy)
	kingpin.Flag("title", "server title").Default("Go HTTP File Server").StringVar(&gcfg.Title)

	kingpin.Parse()
}

func main() {
	parseFlags()

	ss := NewHTTPStaticServer(gcfg.Root)
	ss.Theme = gcfg.Theme
	ss.Title = gcfg.Title

	if gcfg.Upload {
		ss.EnableUpload()
	}
	if gcfg.PlistProxy != nil {
		gcfg.PlistProxy.Scheme = "https"
		ss.PlistProxy = gcfg.PlistProxy.String()
	}

	var hdlr http.Handler = ss

	hdlr = accesslog.NewLoggingHandler(hdlr, l)

	// HTTP Basic Authentication
	userpass := strings.SplitN(gcfg.HttpAuth, ":", 2)
	if len(userpass) == 2 {
		user, pass := userpass[0], userpass[1]
		hdlr = httpauth.SimpleBasicAuth(user, pass)(hdlr)
	}
	// CORS
	if gcfg.Cors {
		hdlr = handlers.CORS()(hdlr)
	}
	if gcfg.XHeaders {
		hdlr = handlers.ProxyHeaders(hdlr)
	}

	http.Handle("/", hdlr)
	http.HandleFunc("/-/sysinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, _ := json.Marshal(map[string]interface{}{
			"version": VERSION,
		})
		w.Write(data)
	})

	log.Printf("Listening on addr: %s\n", strconv.Quote(gcfg.Addr))

	var err error
	if gcfg.Key != "" && gcfg.Cert != "" {
		err = http.ListenAndServeTLS(gcfg.Addr, gcfg.Cert, gcfg.Key, nil)
	} else {
		err = http.ListenAndServe(gcfg.Addr, nil)
	}
	log.Fatal(err)
}

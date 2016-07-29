package main

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	XProxy     bool
	Upload     bool
	PlistProxy *url.URL
}

type logger struct {
}

func (l logger) Log(record accesslog.LogRecord) {
	log.Printf("%s [code %d] %s", record.Method, record.Status, record.Uri)
}

var (
	gcfg = Configure{}
	l    = logger{}
)

func parseFlags() {
	kingpin.HelpFlag.Short('h')
	kingpin.Flag("root", "root directory").Short('r').Default("./").StringVar(&gcfg.Root)
	kingpin.Flag("addr", "listen address").Short('a').Default(":8000").StringVar(&gcfg.Addr)
	kingpin.Flag("cert", "tls cert.pem path").StringVar(&gcfg.Cert)
	kingpin.Flag("key", "tls key.pem path").StringVar(&gcfg.Key)
	kingpin.Flag("cors", "enable cross-site HTTP request").BoolVar(&gcfg.Cors)
	kingpin.Flag("httpauth", "HTTP basic auth (ex: user:pass)").Default("").StringVar(&gcfg.HttpAuth)
	kingpin.Flag("theme", "web theme, one of <black|green>").Default("black").StringVar(&gcfg.Theme)
	kingpin.Flag("xproxy", "Used when behide proxy like nginx").BoolVar(&gcfg.XProxy)
	kingpin.Flag("upload", "Enable upload support").BoolVar(&gcfg.Upload)
	kingpin.Flag("plistproxy", "IPA Plist file proxy, https needed").Short('p').URLVar(&gcfg.PlistProxy)

	kingpin.Parse()
}

func main() {
	parseFlags()

	ss := NewHTTPStaticServer(gcfg.Root)
	ss.Theme = gcfg.Theme

	log.Println(gcfg.PlistProxy)
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
	if gcfg.XProxy {
		hdlr = handlers.ProxyHeaders(hdlr)
	}

	http.Handle("/", hdlr)

	log.Printf("Listening on addr: %s\n", strconv.Quote(gcfg.Addr))

	var err error
	if gcfg.Key != "" && gcfg.Cert != "" {
		err = http.ListenAndServeTLS(gcfg.Addr, gcfg.Cert, gcfg.Key, nil)
	} else {
		err = http.ListenAndServe(gcfg.Addr, nil)
	}
	log.Fatal(err)
}

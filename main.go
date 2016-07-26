package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/goji/httpauth"
	"github.com/rs/cors"
)

type Configure struct {
	Addr     string
	Root     string
	HttpAuth string
	Cert     string
	Key      string
	Cors     bool
	Theme    string
}

var gcfg = Configure{}

func parseFlags() {
	kingpin.HelpFlag.Short('h')
	kingpin.Flag("addr", "listen address").Short('a').Default(":8000").StringVar(&gcfg.Addr)
	kingpin.Flag("cert", "tls cert.pem path").StringVar(&gcfg.Cert)
	kingpin.Flag("key", "tls key.pem path").StringVar(&gcfg.Key)
	kingpin.Flag("cors", "enable cross-site HTTP request").BoolVar(&gcfg.Cors)
	kingpin.Flag("httpauth", "HTTP basic auth (ex: user:pass)").Default("").StringVar(&gcfg.HttpAuth)
	kingpin.Flag("theme", "web theme, one of <black|green>").Default("green").StringVar(&gcfg.Theme)

	kingpin.Parse()
}

func main() {
	parseFlags()

	var hdlr http.Handler = NewHTTPStaticServer("./", gcfg.Theme)

	// HTTP Basic Authentication
	userpass := strings.SplitN(gcfg.HttpAuth, ":", 2)
	if len(userpass) == 2 {
		user, pass := userpass[0], userpass[1]
		hdlr = httpauth.SimpleBasicAuth(user, pass)(hdlr)
	}
	// CORS
	if gcfg.Cors {
		hdlr = cors.Default().Handler(hdlr)
	}

	http.Handle("/", hdlr)

	// indexContent, _ := Asset("res/index.tmpl.html")
	// log.Println(string(indexContent))
	log.Printf("Listening on addr: %s\n", strconv.Quote(gcfg.Addr))

	var err error
	if gcfg.Key != "" && gcfg.Cert != "" {
		err = http.ListenAndServeTLS(gcfg.Addr, gcfg.Cert, gcfg.Key, nil)
	} else {
		err = http.ListenAndServe(gcfg.Addr, nil)
	}
	log.Fatal(err)
}

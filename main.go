package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/alecthomas/kingpin"
)

type Configure struct {
	Addr     string
	Root     string
	HttpAuth string
	Cert     string
	Key      string
}

var gcfg = Configure{}

func parseFlags() {
	kingpin.HelpFlag.Short('h')
	kingpin.Flag("addr", "listen address").Short('a').Default(":8000").StringVar(&gcfg.Addr)
	kingpin.Flag("cert", "tls cert.pem path").StringVar(&gcfg.Cert)
	kingpin.Flag("key", "tls key.pem path").StringVar(&gcfg.Key)

	kingpin.Parse()
}

func main() {
	parseFlags()
	ss := NewHTTPStaticServer("./")
	http.Handle("/", ss)

	log.Printf("Listening on addr: %s\n", strconv.Quote(gcfg.Addr))

	var err error
	if gcfg.Key != "" && gcfg.Cert != "" {
		err = http.ListenAndServeTLS(gcfg.Addr, gcfg.Cert, gcfg.Key, nil)
	} else {
		err = http.ListenAndServe(gcfg.Addr, nil)
	}
	log.Fatal(err)
}

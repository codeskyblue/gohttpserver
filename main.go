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
	kingpin.Parse()
}

func main() {
	parseFlags()
	ss := NewHTTPStaticServer("/")

	log.Printf("Listening on addr: %s\n", strconv.Quote(gcfg.Addr))
	log.Fatal(http.ListenAndServe(gcfg.Addr, ss))
}

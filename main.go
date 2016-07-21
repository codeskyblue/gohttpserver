package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/kataras/iris"
)

type Configure struct {
	Addr     string
	Root     string
	HttpAuth string
	Cert     string
	Key      string
}

var gcfg = Configure{}

func main() {
	kingpin.HelpFlag.Short('h')
	kingpin.Flag("addr", "listen address").Short('a').Default(":8000").StringVar(&gcfg.Addr)
	kingpin.Parse()

	iris.Get("/hi", func(ctx *iris.Context) {
		ctx.Write("Hi %s", "iris")
	})
	iris.Listen(gcfg.Addr)
}

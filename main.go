package main

import (
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/kataras/iris"
	"github.com/kataras/iris/utils"
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

	iris.Get("/*file", func(ctx *iris.Context) {
		requestPath := ctx.Param("file")
		path := strings.Replace(requestPath, "/", utils.PathSeparator, -1)

		if !utils.DirectoryExists(path) {
			ctx.NotFound()
			return
		}
		ctx.ServeFile(path, false)
	})

	iris.Listen(gcfg.Addr)
}

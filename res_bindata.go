// +build bindata

package main

import (
	"html/template"
	"net/http"
)

var tmpl *template.Template

func init() {
	http.Handle("/-/res/", http.StripPrefix("/-/res/", http.FileServer(assetFS())))

	indexContent, _ := Asset("res/index.tmpl.html")
	tmpl = template.Must(template.New("t").Delims("[[", "]]").Parse(string(indexContent)))
}

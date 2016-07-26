// +build !bindata

package main

import (
	"html/template"
	"io/ioutil"
	"net/http"
)

var tmpl *template.Template

func init() {
	http.Handle("/-/res/", http.StripPrefix("/-/res/", http.FileServer(http.Dir("./res"))))

	indexContent, _ := ioutil.ReadFile("./res/index.tmpl.html")
	tmpl = template.Must(template.New("t").Delims("[[", "]]").Parse(string(indexContent)))
}

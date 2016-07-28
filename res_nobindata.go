// +build !bindata

package main

import (
	"io/ioutil"
	"log"
	"net/http"
)

func init() {
	http.Handle("/-/res/", http.StripPrefix("/-/res/", http.FileServer(http.Dir("./res"))))

	for name, path := range templates {
		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		ParseTemplate(name, string(content))
	}
}

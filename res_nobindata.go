// +build !bindata

package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func init() {
	selfDir := filepath.Dir(os.Args[0])
	resDir := filepath.Join(selfDir, "./res")
	http.Handle("/-/res/", http.StripPrefix("/-/res/", http.FileServer(http.Dir(resDir))))

	for name, path := range templates {
		content, err := ioutil.ReadFile(filepath.Join(selfDir, path))
		if err != nil {
			log.Fatal(err)
		}
		ParseTemplate(name, string(content))
	}
}

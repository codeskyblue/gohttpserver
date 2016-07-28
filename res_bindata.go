// +build bindata

package main

import (
	"log"
	"net/http"
)

func init() {
	http.Handle("/-/res/", http.StripPrefix("/-/res/", http.FileServer(assetFS())))

	for name, path := range templates {
		data, err := Asset(path)
		if err != nil {
			log.Fatal(err)
		}
		ParseTemplate(name, string(data))
	}
}

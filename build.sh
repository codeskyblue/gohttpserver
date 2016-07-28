#!/bin/bash -

set -eu

build() {
	echo "$1 $2 ..."
	GOOS=$1 GOARCH=$2 go build -tags bindata -o dist/gohttpserver-${3:-""}
}

go-bindata-assetfs -tags bindata res/...

build darwin amd64 mac-amd64
build linux amd64 linux-amd64
build linux 386 linux-386
build windows amd64 win-amd64.exe

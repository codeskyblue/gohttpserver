#!/bin/bash -

set -eu

VERSION=$(git describe --abbrev=0 --tags)
REVCNT=$(git rev-list --count HEAD)
DEVCNT=$(git rev-list --count $VERSION)
if test $REVCNT != $DEVCNT
then
	VERSION="$VERSION.dev$(expr $REVCNT - $DEVCNT)"
fi
echo "VER: $VERSION"

GITCOMMIT=$(git rev-parse HEAD)
BUILDTIME=$(date -u +%Y/%m/%d-%H:%M:%S)

LDFLAGS="-X main.VERSION=$VERSION -X main.BUILDTIME=$BUILDTIME -X main.GITCOMMIT=$GITCOMMIT"
if test -n "$EX_LDFLAGS"
then
	LDFLAGS="$LDFLAGS $EX_LDFLAGS"
fi

build() {
	echo "$1 $2 ..."
	GOOS=$1 GOARCH=$2 go build \
		-tags bindata \
		-ldflags "$LDFLAGS" \
		-o dist/gohttpserver-${3:-""}
}

go-bindata-assetfs -tags bindata res/...

build darwin amd64 mac-amd64
build linux amd64 linux-amd64
build linux 386 linux-386
build windows amd64 win-amd64.exe

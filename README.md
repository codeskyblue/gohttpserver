# gohttpserver
Make the best HTTP File Server. Rewrite from https://github.com/codeskyblue/gohttp with golang+vue

## Notes
If using go1.5, ensure you set GO15VENDOREXPERIMENT=1

## Features
1. [x] Support QRCode code generate
1. [x] Breadcrumb path quick change
1. [x] All assets package to Standalone binary
1. [x] Different file type different icon
1. [x] Support show or hide hidden files
1. [ ] Upload support
1. [ ] README.md preview
1. [ ] Code preview
1. [x] HTTP Basic Auth
1. [ ] \.htaccess support
1. [x] Partial reload pages when directory change
1. [x] When only one dir under dir, path will combine two together
1. [x] Directory zip download
1. [ ] Apple ipa auto generate .plist file, qrcode can be recognized by iphone (Require https)
1. [ ] Support modify the index page
1. [ ] Download count statistics
1. [x] CORS enabled
1. [ ] Offline download
1. [ ] Edit file support
1. [ ] Global file search
1. [x] Hidden work `download` and `qrcode` in small screen
1. [ ] Theme select support

## Installation
```
go get -v github.com/codeskyblue/gohttpserver
```

## Usage
Listen port 8000 on all interface

```
./gohttpserver --addr :8000
```

## FAQ
- [How to generate self signed certificate with openssl](http://stackoverflow.com/questions/10175812/how-to-create-a-self-signed-certificate-with-openssl)

## Developer Guide
Depdencies are managed by godep

```sh
go get -v github.com/tools/godep
go get github.com/jteeuwen/go-bindata/...
go get github.com/elazarl/go-bindata-assetfs/...
```

## How to build single binary release
```sh
go-bindata-assetfs -tags bindata res/...
go build -tags bindata
```

That's all. ^_^

## Reference Web sites

* <https://vuejs.org.cn/>
* Icon from <http://www.easyicon.net/558394-file_explorer_icon.html>
* <https://github.com/elazarl/go-bindata-assetfs>
* Code Highlight <https://craig.is/making/rainbows>
* Markdown-JS <https://github.com/evilstreak/markdown-js>

## LICENSE
This project is under license [MIT](LICENSE)

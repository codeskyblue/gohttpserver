# gohttp-vue
Rewrite https://github.com/codeskyblue/gohttp with golang+vue

## Not ready yet.

## Notes
If using go1.5, ensure you set GO15VENDOREXPERIMENT=1

## Features
1. [x] Support QRCode code generate
1. [ ] All assets package to Standalone binary
1. [x] Different file type different icon
1. [ ] Support show or hide hidden files
1. [ ] Upload support
1. [ ] README.md preview
1. [x] HTTP Basic Auth
1. [ ] \.htaccess support
1. [ ] Partial reload pages when directory change
1. [ ] When only one dir under dir, path will combine two together
1. [ ] Directory zip download
1. [ ] Code preview
1. [ ] Apple ipa auto generate .plist file, qrcode can be recognized by iphone (Require https)
1. [ ] Support modify the index page
1. [ ] Download count statistics
1. [ ] CORS enabled
1. [ ] Offline download
1. [ ] Edit file support
1. [ ] Global file search
1. [ ] Hidden work `download` and `qrcode` in small screen

## Installation
```
go get -v github.com/codeskyblue/gohttp-vue
```

## Usage
Listen port 8000 on all interface

```
./gohttp-vue --addr :8000
```

## FAQ
- [How to generate self signed certificate with openssl](http://stackoverflow.com/questions/10175812/how-to-create-a-self-signed-certificate-with-openssl)

## Developer Guide
Depdencies are managed by godep

```
go get -v github.com/tools/godep
```

Reference Web sites

* <https://vuejs.org.cn/>
* Icon from <http://www.easyicon.net/558394-file_explorer_icon.html>
## LICENSE
This project is under license [MIT](LICENSE)

# gohttpserver
[![Build Status](https://travis-ci.org/codeskyblue/gohttpserver.svg?branch=master)](https://travis-ci.org/codeskyblue/gohttpserver)

Make the best HTTP File Server. Better UI, upload support, apple&android install package qrcode generate.

[Demo site](https://gohttpserver.herokuapp.com/)

- 目标: 做最好的HTTP文件服务器
- 功能: 人性化的UI体验，文件的上传支持，安卓和苹果安装包的二维码直接生成。

**Binary** can be download from [github releases](https://github.com/codeskyblue/gohttpserver/releases/)

## Notes
If using go1.5, ensure you set GO15VENDOREXPERIMENT=1

Upload size now limited to 1G

## Screenshots
![screen](testdata/filetypes/gohttpserver.gif)

## Features
1. [x] Support QRCode code generate
1. [x] Breadcrumb path quick change
1. [x] All assets package to Standalone binary
1. [x] Different file type different icon
1. [x] Support show or hide hidden files
1. [x] Upload support (for security reason, you need enabled it by option `--upload`)
1. [x] README.md preview
1. [x] HTTP Basic Auth
1. [x] Partial reload pages when directory change
1. [x] When only one dir under dir, path will combine two together
1. [x] Directory zip download
1. [x] Apple ipa auto generate .plist file, qrcode can be recognized by iphone (Require https)
1. [x] Plist proxy
1. [ ] Download count statistics
1. [x] CORS enabled
1. [ ] Offline download
1. [ ] Code file preview
1. [ ] Edit file support
1. [x] Global file search
1. [x] Hidden work `download` and `qrcode` in small screen
1. [x] Theme select support
1. [x] OK to working behide Nginx
1. [ ] \.ghs.yml support (like \.htaccess)
1. [ ] Calculate md5sum and sha
1. [ ] Folder upload
1. [ ] Support sort by size or modified time
1. [x] Add version info into index page
1. [ ] Add api `/-/info/some.(apk|ipa)` to get detail info
1. [x] Auto tag version
1. [x] Custom title support
1. [x] Support setting from conf file

## Installation
```
go get -v github.com/codeskyblue/gohttpserver
cd $GOPATH/src/github.com/codeskyblue/gohttpserver
go build && ./gohttpserver
```

## Usage
Listen port 8000 on all interface, and enable upload

```
./gohttpserver -r ./ --addr :8000 --upload
```

## Advanced usage
Support update access rule if there is a file named `.ghs.yml` under directory. `.ghs.yml` example

```yaml
---
upload: false
```

For example, if there is such file under directory `foo`, directory `foo` can not be uploaded, while `bar` can.

```
root -
  |-- foo
  |    |-- .ghs.yml
  |    `-- world.txt 
  `-- bar
       `-- hello.txt
```

Use config file. specfied with `--conf`, see [example config.yml](testdata/config.yml). Note that command line option can overwrite conf in `config.yml`

### ipa plist proxy
This is used for server which not https enabled. default use <https://plistproxy.herokuapp.com/plist>

```
./gohttpserver --plistproxy=https://someproxyhost.com/
```

Proxy web site should have ability

```sh
$ http POST https://proxyhost.com/plist < app.plist
{
	"key": "18f99211"
}
$ http GET https://proxyhost.com/plist/18f99211
# show the app.plist content
```

### Upload with CURL
For example, upload a file named `foo.txt` to directory `somedir`

PS: max upload size limited to 1G (hard coded)

```sh
$ curl -F file=@foo.txt localhost:8000/somedir
```

## FAQ
- [How to generate self signed certificate with openssl](http://stackoverflow.com/questions/10175812/how-to-create-a-self-signed-certificate-with-openssl)

### How the search works
The search algorithm follow the search engine google. keywords are seperated with space, words with prefix `-` will be excluded.

1. `hello world` means must contains `hello` and `world`
1. `hello -world` means must contains `hello` but not contains `world`

## Developer Guide
Depdencies are managed by godep

```sh
go get -v github.com/tools/godep
go get github.com/jteeuwen/go-bindata/...
go get github.com/elazarl/go-bindata-assetfs/...
```

Theme are all defined in [res/themes](res/themes) directory. Now only two, black and green.

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
* Markdown-JS <https://github.com/showdownjs/showdown>
* <https://github.com/sindresorhus/github-markdown-css>
* <http://www.gorillatoolkit.org/pkg/handlers>
* <http://www.dropzonejs.com/>

## History
The first version is <https://github.com/codeskyblue/gohttp>

## LICENSE
This project is under license [MIT](LICENSE)

jQuery('#qrcodeCanvas').qrcode({
    text: "http://jetienne.com/"
});

function getExtention(fname) {
    return fname.slice((fname.lastIndexOf(".") - 1 >>> 0) + 2);
}

function pathJoin(parts, sep) {
    var separator = sep || '/';
    var replace = new RegExp(separator + '{1,}', 'g');
    return parts.join(separator).replace(replace, separator);
}

function getQueryString(name) {
    var reg = new RegExp("(^|&)" + name + "=([^&]*)(&|$)");
    var r = decodeURI(window.location.search).substr(1).match(reg);
    if (r != null) return r[2].replace(/\+/g, ' ');
    return null;
}

var vm = new Vue({
    el: "#app",
    data: {
        message: "Hello vue.js",
        breadcrumb: [],
        showHidden: false,
        previewFile: null,
        version: "loading",
        mtimeTypeFromNow: false, // or fromNow
        auth: {},
        search: getQueryString("search"),
        files: [{
            name: "loading ...",
            path: "",
            size: "...",
            type: "dir",
        }]
    },
    computed: {
        computedFiles: function() {
            var that = this;
            this.previewFile = null;

            var files = this.files.filter(function(f) {
                if (f.name == 'README.md') {
                    that.previewFile = {
                        name: f.name,
                        path: f.path,
                        size: f.size,
                        type: 'markdown',
                        contentHTML: '',
                    }
                }
                if (!that.showHidden && f.name.slice(0, 1) === '.') {
                    return false;
                }
                return true;
            });
            // console.log(this.previewFile)
            if (this.previewFile) {
                var name = this.previewFile.name; // For now only README.md
                console.log(pathJoin([location.pathname, 'README.md']))
                $.ajax({
                    url: pathJoin([location.pathname, 'README.md']),
                    method: 'GET',
                    success: function(res) {
                        var converter = new showdown.Converter({
                            tables: true,
                            omitExtraWLInCodeBlocks: true,
                            parseImgDimensions: true,
                            simplifiedAutoLink: true,
                            literalMidWordUnderscores: true,
                            tasklists: true,
                            ghCodeBlocks: true,
                            smoothLivePreview: true,
                        });

                        var html = converter.makeHtml(res);
                        that.previewFile.contentHTML = html;
                    },
                    error: function(err) {
                        console.log(err)
                    }
                })
            }

            return files;
        },
    },
    methods: {
        formatTime: function(timestamp) {
            var m = moment(timestamp);
            if (this.mtimeTypeFromNow) {
                return m.fromNow();
            }
            return m.format('YYYY-MM-DD HH:mm:ss');
        },
        toggleHidden: function() {
            this.showHidden = !this.showHidden;
        },
        genInstallURL: function(name) {
            if (getExtention(name) == "ipa") {
                urlPath = location.protocol + "//" + pathJoin([location.host, "/-/ipa/link", location.pathname, name]);
                return urlPath;
            }
            return location.protocol + "//" + pathJoin([location.host, location.pathname, name]);
        },
        genQrcode: function(text) {
            var urlPath = this.genInstallURL(text);
            $("#qrcode-title").html(text);
            $("#qrcode-link").attr("href", urlPath);
            $('#qrcodeCanvas').empty().qrcode({
                text: urlPath
            });
            $("#qrcode-modal").modal("show");
        },
        shouldHaveQrcode: function(name) {
            return ['apk', 'ipa'].indexOf(getExtention(name)) !== -1;
        },
        genFileClass: function(f) {
            if (f.type == "dir") {
                if (f.name == '.git') {
                    return 'fa-git-square';
                }
                return "fa-folder-open";
            }
            var ext = getExtention(f.name);
            switch (ext) {
                case "go":
                case "py":
                case "js":
                case "java":
                case "c":
                case "cpp":
                case "h":
                    return "fa-file-code-o";
                case "pdf":
                    return "fa-file-pdf-o";
                case "zip":
                    return "fa-file-zip-o";
                case "mp3":
                case "wav":
                    return "fa-file-audio-o";
                case "jpg":
                case "png":
                case "gif":
                case "jpeg":
                case "tiff":
                    return "fa-file-picture-o";
                case "ipa":
                case "dmg":
                    return "fa-apple";
                case "apk":
                    return "fa-android";
                case "exe":
                    return "fa-windows";
            }
            return "fa-file-text-o"
        },
        clickFileOrDir: function(f, e) {
            if (f.type == "file") {
                return true;
            }
            var reqPath = pathJoin([location.pathname, f.name]);
            loadDirectory(reqPath);
            e.preventDefault()
        },
        changePath: function(reqPath, e) {
            loadDirectory(reqPath);
            e.preventDefault()
        },
        updateBreadcrumb: function() {
            var pathname = decodeURI(location.pathname || "/");
            var parts = pathname.split('/');
            this.breadcrumb = [];
            if (pathname == "/") {
                return this.breadcrumb;
            }
            var i = 2;
            for (; i <= parts.length; i += 1) {
                var name = parts[i - 1];
                var path = parts.slice(0, i).join('/');
                this.breadcrumb.push({
                    name: name + (i == parts.length ? ' /' : ''),
                    path: path
                })
            }
            return this.breadcrumb;
        }
    }
})

window.onpopstate = function(event) {
    var pathname = decodeURI(location.pathname)
    loadFileList()
}

function loadDirectory(reqPath) {
    window.history.pushState({}, "", reqPath);
    loadFileList(reqPath)
}

function loadFileList(pathname) {
    var pathname = pathname || location.pathname;
    // console.log("load filelist:", pathname)
    $.ajax({
        url: pathJoin(["/-/json", pathname]),
        dataType: "json",
        cache: false,
        success: function(res) {
            res.files.sort(function(a, b) {
                var obj2n = function(v) {
                    return v.type == "dir" ? 0 : 1;
                };
                return (obj2n(a) - obj2n(b)) || (a.name > b.name);
            })
            vm.files = res.files;
            vm.auth = res.auth;
        },
        error: function(err) {
            console.error(err)
        },
    });
    vm.updateBreadcrumb();
}

// For page first loading
loadFileList(location.pathname + location.search)

// update version
$.getJSON("/-/sysinfo", function(res) {
    vm.version = res.version;
})

Dropzone.options.myDropzone = {
    paramName: "file",
    maxFilesize: 1024,
    addRemoveLinks: true,
    init: function() {
        this.on("uploadprogress", function(file, progress) {
            console.log("File progress", progress);
        });
        this.on("complete", function(file) {
            console.log("reload file list")
            loadFileList()
        })
    }
}

Vue.filter('fromNow', function(value) {
    return moment(value).fromNow();
})

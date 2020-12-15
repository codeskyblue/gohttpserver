<!DOCTYPE html>
<html>

<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no">
  <title>[[.Title]]</title>
  <link rel="shortcut icon" type="image/png" href="/-/assets/favicon.png" />
  <link rel="stylesheet" type="text/css" href="/-/assets/bootstrap-3.3.5/css/bootstrap.min.css">
  <link rel="stylesheet" type="text/css" href="/-/assets/font-awesome-4.6.3/css/font-awesome.min.css">
  <link rel="stylesheet" type="text/css" href="/-/assets/css/github-markdown.css">
  <link rel="stylesheet" type="text/css" href="/-/assets/css/dropzone.css">
  <link rel="stylesheet" type="text/css" href="/-/assets/css/scrollUp-image.css">
  <link rel="stylesheet" type="text/css" href="/-/assets/css/style.css">
  <link rel="stylesheet" type="text/css" href="/-/assets/themes/[[.Theme]].css">
</head>

<body id="app">
  <nav class="navbar navbar-default">
    <div class="container">
      <div class="container">
        <div class="navbar-header">
          <button type="button" class="navbar-toggle collapsed" data-toggle="collapse" data-target="#bs-example-navbar-collapse-2">
            <span class="sr-only">Toggle navigation</span>
            <span class="icon-bar"></span>
            <span class="icon-bar"></span>
            <span class="icon-bar"></span>
          </button>
          <a class="navbar-brand" href="/">[[.Title]]</a>
        </div>
        <div class="collapse navbar-collapse" id="bs-example-navbar-collapse-2">
          <ul class="nav navbar-nav">
            <li class="hidden-xs">
              <a href="javascript:void(0)" v-on:click='genQrcode()'>
                View in Phone
                <span class="glyphicon glyphicon-qrcode"></span>
              </a>
            </li>
            [[if eq .AuthType "openid"]]
            <template v-if="!user.email">
              <a href="/-/login" class="btn btn-sm btn-default navbar-btn">
                Sign in <span class="glyphicon glyphicon-user"></span>
              </a>
            </template>
            <template v-else>
              <a href="/-/logout" class="btn btn-sm btn-default navbar-btn">
                <span v-text="user.name"></span>
                <i class="fa fa-sign-out"></i>
              </a>
            </template>
            [[end]]
            [[if eq .AuthType "oauth2-proxy"]]
            <template v-if="!user.email">
                <a href="#" class="btn btn-sm btn-default navbar-btn">
                    Guest <span class="glyphicon glyphicon-user"></span>
                </a>
            </template>
            <template v-else>
                <a href="/-/logout" class="btn btn-sm btn-default navbar-btn">
                    <span v-text="user.name"></span>
                    <i class="fa fa-sign-out"></i>
                </a>
            </template>
            [[end]]
          </ul>
          <form class="navbar-form navbar-right">
            <div class="input-group">
              <input type="text" name="search" class="form-control" placeholder="Search text" v-bind:value="search"
                autofocus>
              <span class="input-group-btn">
                <button class="btn btn-default" type="submit">
                  <span class="glyphicon glyphicon-search"></span>
                </button>
              </span>
            </div>
          </form>
          <ul id="nav-right-bar" class="nav navbar-nav navbar-right">
          </ul>
        </div>
      </div>
    </div>
  </nav>
  <div class="container">
    <div class="col-md-12">
      <ol class="breadcrumb">
        <li>
          <a v-on:click='changePath("/", $event)' href="/"><i class="fa fa-home"></i></a>
        </li>
        <li v-for="bc in breadcrumb.slice(0, breadcrumb.length-1)">
          <a v-on:click='changePath(bc.path, $event)' href="{{bc.path}}">{{bc.name}}</a>
        </li>
        <li v-if="breadcrumb.length >= 1">
          {{breadcrumb.slice(-1)[0].name}}
        </li>
      </ol>
      <table class="table table-hover" v-if="!previewMode">
        <thead>
          <tr>
            <td colspan=4>
              <!-- <button class="btn btn-xs btn-default" v-on:click='toggleHidden()'>
                Back <i class="fa" v-bind:class='showHidden ? "fa-eye" : "fa-eye-slash"'></i>
              </button> -->
              <div>
                <button class="btn btn-xs btn-default" onclick="history.back()">
                  Back <i class="fa fa-arrow-left"></i>
                </button>
                <button class="btn btn-xs btn-default" v-on:click='toggleHidden()'>
                  Hidden <i class="fa" v-bind:class='showHidden ? "fa-eye" : "fa-eye-slash"'></i>
                </button>
                <button class="btn btn-xs btn-default" v-show="auth.upload" data-toggle="modal" data-target="#upload-modal">
                  Upload <i class="fa fa-upload"></i>
                </button>
                <button class="btn btn-xs btn-default" v-show="auth.delete" @click="makeDirectory">
                  New Folder <i class="fa fa-folder"></i>
                </button>
              </div>
            </td>
          </tr>
          <tr>
            <th>Name</th>
            <th>Size</th>
            <th class="hidden-xs">
              <span style="cursor: pointer" v-on:click='mtimeTypeFromNow = !mtimeTypeFromNow'>ModTime</span>
            </th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="f in computedFiles">
            <td>
              <a v-on:click='clickFileOrDir(f, $event)' href="/{{f.path + (f.type == 'dir' ? '' : '')}}">
                <!-- ?raw=false -->
                <i style="padding-right: 0.5em" class="fa" v-bind:class='genFileClass(f)'></i> {{f.name}}
              </a>
              <!-- for search -->
              <button v-show="f.type == 'file' && f.name.indexOf('/') >= 0" class="btn btn-default btn-xs" @click="changeParentDirectory(f.path)">
                <i class="fa fa-folder-open-o"></i>
              </button>
            </td>
            <td><span v-if="f.type == 'dir'">~</span> {{f.size | formatBytes}}</td>
            <td class="hidden-xs">{{formatTime(f.mtime)}}</td>
            <td style="text-align: left">
              <template v-if="f.type == 'dir'">
                <a class="btn btn-default btn-xs" href="/{{f.path}}/?op=archive">
                  <span class="hidden-xs">Archive</span> Zip
                  <span class="glyphicon glyphicon-download-alt"></span>
                </a>
                <button class="btn btn-default btn-xs" v-on:click="showInfo(f)">
                    <span class="glyphicon glyphicon-info-sign"></span>
                </button>
                <button class="btn btn-default btn-xs" v-if="auth.delete" v-on:click="deletePathConfirm(f, $event)">
                  <span style="color:#CC3300" class="glyphicon glyphicon-trash"></span>
                </button>
              </template>
              <template v-if="f.type == 'file'">
                <a class="btn btn-default btn-xs hidden-xs" href="{{genDownloadURL(f)}}">
                  <span class="hidden-xs">Download</span>
                  <span class="glyphicon glyphicon-download-alt"></span>
                </a>
                <button class="btn btn-default btn-xs bstooltip" data-trigger="manual" data-title="Copied!"
                  data-clipboard-text="{{genDownloadURL(f)}}">
                  <i class="fa fa-copy"></i>
                </button>
                <button class="btn btn-default btn-xs" v-on:click="showInfo(f)">
                  <span class="glyphicon glyphicon-info-sign"></span>
                </button>
                <button class="btn btn-default btn-xs hidden-xs" v-on:click="genQrcode(f.name)">
                  <span v-if="shouldHaveQrcode(f.name)">QRCode</span>
                  <span class="glyphicon glyphicon-qrcode"></span>
                </button>
                <a class="btn btn-default btn-xs visible-xs" v-if="shouldHaveQrcode(f.name)" href="{{genInstallURL(f.name)}}">
                  Install <i class="fa fa-cube"></i>
                </a>
                <button class="btn btn-default btn-xs" v-if="auth.delete" v-on:click="deletePathConfirm(f, $event)">
                  <span style="color:#CC3300" class="glyphicon glyphicon-trash"></span>
                </button>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div class="col-md-12" id="preview" v-if="preview.filename">
      <div class="panel panel-default">
        <div class="panel-heading">
          <h3 class="panel-title" style="font-weight: normal">
            <i class="fa" v-bind:class='genFileClass(previewFile)'></i> {{preview.filename}}
          </h3>
        </div>
        <div class="panel-body">
          <article class="markdown-body">{{{preview.contentHTML }}}
          </article>
        </div>
      </div>
    </div>
    <div class="col-md-12" id="content">
      <!-- Small qrcode modal -->
      <div id="qrcode-modal" class="modal fade" tabindex="-1" role="dialog">
        <div class="modal-dialog">
          <div class="modal-content">
            <div class="modal-header">
              <button type="button" class="close" data-dismiss="modal" aria-label="Close"><span aria-hidden="true">&times;</span></button>
              <h4 class="modal-title">
                <span id="qrcode-title"></span>
                <a style="font-size: 0.6em" href="#" id="qrcode-link">[view]</a>
              </h4>
            </div>
            <div class="modal-body clearfix">
              <div id="qrcodeCanvas" class="pull-left"></div>
              <div id="qrcodeRight" class="pull-left">
                <p>
                  <a href="#">下载链接</a>
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
      <!-- Upload modal-->
      <div id="upload-modal" class="modal fade" tabindex="-1" role="dialog">
        <div class="modal-dialog">
          <div class="modal-content">
            <div class="modal-header">
              <button type="button" class="close" data-dismiss="modal" aria-label="Close"><span aria-hidden="true">&times;</span></button>
              <h4 class="modal-title">
                <i class="fa fa-upload"></i> File upload
              </h4>
            </div>
            <div class="modal-body">
              <form action="#" class="dropzone" id="upload-form"></form>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-default" @click="removeAllUploads">RemoveAll</button>
              <button type="button" class="btn btn-default" data-dismiss="modal">Close</button>
            </div>
          </div>
        </div>
      </div>
      <!-- File info modal -->
      <div id="file-info-modal" class="modal fade" tabindex="-1" role="dialog">
        <div class="modal-dialog">
          <div class="modal-content">
            <div class="modal-header">
              <button type="button" class="close" data-dismiss="modal" aria-label="Close"><span aria-hidden="true">&times;</span></button>
              <h4 class="modal-title">
                <span id="file-info-title"></span>
              </h4>
            </div>
            <div class="modal-body">
              <pre id="file-info-content"></pre>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div class="col-md-12">
      <div id="footer" class="pull-right" style="margin: 2em 1em">
        <a href="https://github.com/codeskyblue/gohttpserver">gohttpserver (ver:{{version}})</a>, written by <a href="https://github.com/codeskyblue">codeskyblue</a>.
        Copyright 2016-2018. go1.10
      </div>
    </div>
  </div>
  <script src="/-/assets/js/jquery-3.1.0.min.js"></script>
  <script src="/-/assets/js/jquery.qrcode.js"></script>
  <script src="/-/assets/js/jquery.scrollUp.min.js"></script>
  <script src="/-/assets/js/qrcode.js"></script>
  <script src="/-/assets/js/vue-1.0.min.js"></script>
  <script src="/-/assets/js/showdown-1.6.4.min.js"></script>
  <script src="/-/assets/js/moment.min.js"></script>
  <script src="/-/assets/js/dropzone.js"></script>
  <script src="/-/assets/js/underscore-min.js"></script>
  <script src="/-/assets/js/clipboard-1.5.12.min.js"></script>
  <script src="/-/assets/bootstrap-3.3.5/js/bootstrap.min.js"></script>
  <script src='/-/assets/[["js/index.js" | urlhash ]]'></script>
  <!-- <script src="/-/assets/js/index.js"></script> -->
  [[if .GoogleTrackerID ]]
  <script>
    (function (i, s, o, g, r, a, m) {
      i['GoogleAnalyticsObject'] = r;
      i[r] = i[r] || function () {
        (i[r].q = i[r].q || []).push(arguments)
      }, i[r].l = 1 * new Date();
      a = s.createElement(o),
        m = s.getElementsByTagName(o)[0];
      a.async = 1;
      a.src = g;
      m.parentNode.insertBefore(a, m)
    })(window, document, 'script', 'https://www.google-analytics.com/analytics.js', 'ga');

    ga('create', '[[.GoogleTrackerID]]', 'auto');
    ga('send', 'pageview');
  </script> [[ end ]]
</body>

</html>
package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/alecthomas/kingpin"
	accesslog "github.com/codeskyblue/go-accesslog"
	"github.com/go-yaml/yaml"
	"github.com/goji/httpauth"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type Configure struct {
	Conf            *os.File `yaml:"-"`
	Addr            string   `yaml:"addr"`
	Port            int      `yaml:"port"`
	Root            string   `yaml:"root"`
	Prefix          string   `yaml:"prefix"`
	HTTPAuth        string   `yaml:"httpauth"`
	Cert            string   `yaml:"cert"`
	Key             string   `yaml:"key"`
	Theme           string   `yaml:"theme"`
	XHeaders        bool     `yaml:"xheaders"`
	Upload          bool     `yaml:"upload"`
	Delete          bool     `yaml:"delete"`
	PlistProxy      string   `yaml:"plistproxy"`
	Title           string   `yaml:"title"`
	Debug           bool     `yaml:"debug"`
	GoogleTrackerID string   `yaml:"google-tracker-id"`
	Auth            struct {
		Type   string   `yaml:"type"` // openid|http|github
		OpenID string   `yaml:"openid"`
		HTTP   []string `yaml:"http"`
		ID     string   `yaml:"id"`     // for oauth2
		Secret string   `yaml:"secret"` // for oauth2
	} `yaml:"auth"`
	DeepPathMaxDepth int  `yaml:"deep-path-max-depth"`
	NoIndex          bool `yaml:"no-index"`
}

type httpLogger struct{}

func (l httpLogger) Log(record accesslog.LogRecord) {
	log.Printf("%s - %s %d %s", record.Ip, record.Method, record.Status, record.Uri)
}

var (
	defaultPlistProxy = "https://plistproxy.herokuapp.com/plist"
	defaultOpenID     = "https://login.netease.com/openid"
	gcfg              = Configure{}
	logger            = httpLogger{}

	VERSION   = "unknown"
	BUILDTIME = "unknown time"
	GITCOMMIT = "unknown git commit"
	SITE      = "https://github.com/codeskyblue/gohttpserver"
)

func versionMessage() string {
	t := template.Must(template.New("version").Parse(`GoHTTPServer
  Version:        {{.Version}}
  Go version:     {{.GoVersion}}
  OS/Arch:        {{.OSArch}}
  Git commit:     {{.GitCommit}}
  Built:          {{.Built}}
  Site:           {{.Site}}`))
	buf := bytes.NewBuffer(nil)
	t.Execute(buf, map[string]interface{}{
		"Version":   VERSION,
		"GoVersion": runtime.Version(),
		"OSArch":    runtime.GOOS + "/" + runtime.GOARCH,
		"GitCommit": GITCOMMIT,
		"Built":     BUILDTIME,
		"Site":      SITE,
	})
	return buf.String()
}

func parseFlags() error {
	// initial default conf
	gcfg.Root = "./"
	gcfg.Port = 8000
	gcfg.Addr = ""
	gcfg.Theme = "black"
	gcfg.PlistProxy = defaultPlistProxy
	gcfg.Auth.OpenID = defaultOpenID
	gcfg.GoogleTrackerID = "UA-81205425-2"
	gcfg.Title = "Go HTTP File Server"
	gcfg.DeepPathMaxDepth = 5
	gcfg.NoIndex = false

	kingpin.HelpFlag.Short('h')
	kingpin.Version(versionMessage())
	kingpin.Flag("conf", "config file path, yaml format").FileVar(&gcfg.Conf)
	kingpin.Flag("root", "root directory, default ./").Short('r').StringVar(&gcfg.Root)
	kingpin.Flag("prefix", "url prefix, eg /foo").StringVar(&gcfg.Prefix)
	kingpin.Flag("port", "listen port, default 8000").IntVar(&gcfg.Port)
	kingpin.Flag("addr", "listen address, eg 127.0.0.1:8000").Short('a').StringVar(&gcfg.Addr)
	kingpin.Flag("cert", "tls cert.pem path").StringVar(&gcfg.Cert)
	kingpin.Flag("key", "tls key.pem path").StringVar(&gcfg.Key)
	kingpin.Flag("auth-type", "Auth type <http|openid>").StringVar(&gcfg.Auth.Type)
	kingpin.Flag("auth-http", "HTTP basic auth (ex: user:pass)").StringsVar(&gcfg.Auth.HTTP)
	kingpin.Flag("auth-openid", "OpenID auth identity url").StringVar(&gcfg.Auth.OpenID)
	kingpin.Flag("theme", "web theme, one of <black|green>").StringVar(&gcfg.Theme)
	kingpin.Flag("upload", "enable upload support").BoolVar(&gcfg.Upload)
	kingpin.Flag("delete", "enable delete support").BoolVar(&gcfg.Delete)
	kingpin.Flag("xheaders", "used when behide nginx").BoolVar(&gcfg.XHeaders)
	kingpin.Flag("debug", "enable debug mode").BoolVar(&gcfg.Debug)
	kingpin.Flag("plistproxy", "plist proxy when server is not https").Short('p').StringVar(&gcfg.PlistProxy)
	kingpin.Flag("title", "server title").StringVar(&gcfg.Title)
	kingpin.Flag("google-tracker-id", "set to empty to disable it").StringVar(&gcfg.GoogleTrackerID)
	kingpin.Flag("deep-path-max-depth", "set to -1 to not combine dirs").IntVar(&gcfg.DeepPathMaxDepth)
	kingpin.Flag("no-index", "disable indexing").BoolVar(&gcfg.NoIndex)

	kingpin.Parse() // first parse conf

	if gcfg.Conf != nil {
		defer func() {
			kingpin.Parse() // command line priority high than conf
		}()
		ymlData, err := ioutil.ReadAll(gcfg.Conf)
		if err != nil {
			return err
		}
		return yaml.Unmarshal(ymlData, &gcfg)
	}
	return nil
}

func fixPrefix(prefix string) string {
	prefix = regexp.MustCompile(`/*$`).ReplaceAllString(prefix, "")
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if prefix == "/" {
		prefix = ""
	}
	return prefix
}

func cors(next http.Handler) http.Handler {
	// access control and CORS middleware
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		next.ServeHTTP(w, r)
	})
}

func multiBasicAuth(auths []string) func(http.Handler) http.Handler {
	userPassMap := make(map[string]string)
	for _, auth := range auths {
		userpass := strings.SplitN(auth, ":", 2)
		if len(userpass) == 2 {
			userPassMap[userpass[0]] = userpass[1]
		}
	}
	return httpauth.BasicAuth(httpauth.AuthOptions{
		Realm: "Restricted",
		AuthFunc: func(user, pass string, request *http.Request) bool {
			password, ok := userPassMap[user]
			if !ok {
				return false
			}
			givenPass := sha256.Sum256([]byte(pass))
			requiredPass := sha256.Sum256([]byte(password))
			return subtle.ConstantTimeCompare(givenPass[:], requiredPass[:]) == 1
		},
	})
}

func main() {
	if err := parseFlags(); err != nil {
		log.Fatal(err)
	}
	if gcfg.Debug {
		data, _ := yaml.Marshal(gcfg)
		fmt.Printf("--- config ---\n%s\n", string(data))
	}
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	// make sure prefix matches: ^/.*[^/]$
	gcfg.Prefix = fixPrefix(gcfg.Prefix)
	if gcfg.Prefix != "" {
		log.Printf("url prefix: %s", gcfg.Prefix)
	}

	ss := NewHTTPStaticServer(gcfg.Root, gcfg.NoIndex)
	ss.Prefix = gcfg.Prefix
	ss.Theme = gcfg.Theme
	ss.Title = gcfg.Title
	ss.GoogleTrackerID = gcfg.GoogleTrackerID
	ss.Upload = gcfg.Upload
	ss.Delete = gcfg.Delete
	ss.AuthType = gcfg.Auth.Type
	ss.DeepPathMaxDepth = gcfg.DeepPathMaxDepth

	if gcfg.PlistProxy != "" {
		u, err := url.Parse(gcfg.PlistProxy)
		if err != nil {
			log.Fatal(err)
		}
		u.Scheme = "https"
		ss.PlistProxy = u.String()
	}
	if ss.PlistProxy != "" {
		log.Printf("plistproxy: %s", strconv.Quote(ss.PlistProxy))
	}

	var hdlr http.Handler = ss

	hdlr = accesslog.NewLoggingHandler(hdlr, logger)

	// HTTP Basic Authentication
	switch gcfg.Auth.Type {
	case "http":
		hdlr = multiBasicAuth(gcfg.Auth.HTTP)(hdlr)
	case "openid":
		handleOpenID(gcfg.Auth.OpenID, false) // FIXME(ssx): set secure default to false
		// case "github":
		// 	handleOAuth2ID(gcfg.Auth.Type, gcfg.Auth.ID, gcfg.Auth.Secret) // FIXME(ssx): set secure default to false
	case "oauth2-proxy":
		handleOauth2()
	}

	// CORS
	hdlr = cors(hdlr)

	if gcfg.XHeaders {
		hdlr = handlers.ProxyHeaders(hdlr)
	}

	mainRouter := mux.NewRouter()
	router := mainRouter
	if gcfg.Prefix != "" {
		router = mainRouter.PathPrefix(gcfg.Prefix).Subrouter()
		mainRouter.Handle(gcfg.Prefix, hdlr)
		mainRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, gcfg.Prefix, http.StatusTemporaryRedirect)
		})
	}

	router.PathPrefix("/-/assets/").Handler(http.StripPrefix(gcfg.Prefix+"/-/", http.FileServer(Assets)))
	router.HandleFunc("/-/sysinfo", func(w http.ResponseWriter, r *http.Request) {
		data, _ := json.Marshal(map[string]interface{}{
			"version": VERSION,
		})
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Write(data)
	})
	router.PathPrefix("/").Handler(hdlr)

	if gcfg.Addr == "" {
		gcfg.Addr = fmt.Sprintf(":%d", gcfg.Port)
	}
	if !strings.Contains(gcfg.Addr, ":") {
		gcfg.Addr = ":" + gcfg.Addr
	}
	_, port, _ := net.SplitHostPort(gcfg.Addr)
	log.Printf("listening on %s, local address http://%s:%s\n", strconv.Quote(gcfg.Addr), getLocalIP(), port)

	srv := &http.Server{
		Handler: mainRouter,
		Addr:    gcfg.Addr,
	}

	var err error
	if gcfg.Key != "" && gcfg.Cert != "" {
		err = srv.ListenAndServeTLS(gcfg.Cert, gcfg.Key)
	} else {
		err = srv.ListenAndServe()
	}
	log.Fatal(err)
}

package main

import (
	"encoding/gob"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	openid "github.com/codeskyblue/openid-go"
	"github.com/gorilla/sessions"
)

var (
	nonceStore         = openid.NewSimpleNonceStore()
	discoveryCache     = openid.NewSimpleDiscoveryCache()
	store              = sessions.NewCookieStore([]byte("something-very-secret"))
	defaultSessionName = "ghs-session"
)

type UserInfo struct {
	Id       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	NickName string `json:"nickName"`
}

type M map[string]interface{}

func init() {
	gob.Register(&UserInfo{})
	gob.Register(&M{})
}

func handleOpenID(loginUrl string, secure bool) {
	http.HandleFunc("/-/login", func(w http.ResponseWriter, r *http.Request) {
		nextUrl := r.FormValue("next")
		referer := r.Referer()
		if nextUrl == "" && strings.Contains(referer, "://"+r.Host) {
			nextUrl = referer
		}
		scheme := "http"
		if r.URL.Scheme != "" {
			scheme = r.URL.Scheme
		}
		log.Println("Scheme:", scheme)
		if url, err := openid.RedirectURL(loginUrl,
			scheme+"://"+r.Host+"/-/openidcallback?next="+nextUrl, ""); err == nil {
			http.Redirect(w, r, url, 303)
		} else {
			log.Println("Should not got error here:", err)
		}
	})

	http.HandleFunc("/-/openidcallback", func(w http.ResponseWriter, r *http.Request) {
		id, err := openid.Verify("http://"+r.Host+r.URL.String(), discoveryCache, nonceStore)
		if err != nil {
			io.WriteString(w, "Authentication check failed.")
			return
		}
		session, err := store.Get(r, defaultSessionName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		user := &UserInfo{
			Id:       id,
			Email:    r.FormValue("openid.sreg.email"),
			Name:     r.FormValue("openid.sreg.fullname"),
			NickName: r.FormValue("openid.sreg.nickname"),
		}
		session.Values["user"] = user
		if err := session.Save(r, w); err != nil {
			log.Println("session save error:", err)
		}

		nextUrl := r.FormValue("next")
		if nextUrl == "" {
			nextUrl = "/"
		}
		http.Redirect(w, r, nextUrl, 302)
	})

	http.HandleFunc("/-/user", func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, defaultSessionName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		val := session.Values["user"]
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		data, _ := json.Marshal(val)
		w.Write(data)
	})

	http.HandleFunc("/-/logout", func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, defaultSessionName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		delete(session.Values, "user")
		session.Options.MaxAge = -1
		nextUrl := r.FormValue("next")
		_ = session.Save(r, w)
		if nextUrl == "" {
			nextUrl = r.Referer()
		}
		http.Redirect(w, r, nextUrl, 302)
	})
}

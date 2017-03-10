package main

import (
	"context"
	"log"

	"net/url"

	"html/template"
	"io/ioutil"

	"net/http"

	"golang.org/x/oauth2"
)

var tmpls = template.Must(template.ParseGlob("*.html"))

var global struct {
	slack struct {
		conf      *oauth2.Config
		token     *slackToken
		cacheFile string
	}
}

func makeSlakeHandler(apiURL string, params url.Values) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := global.slack.token
		if tok == nil {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("access token for Slack are missing"))
		}

		conf := global.slack.conf
		client := conf.Client(context.Background(), tok.Token)

		if params == nil {
			params = make(url.Values)
		}
		params["token"] = []string{tok.AccessToken}
		resp, err := client.PostForm(apiURL, params)
		if err != nil {
			log.Fatal(err)
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write(data)
		log.Println(r.Method, r.URL.Path)
	}
}

func main() {
	b, err := ioutil.ReadFile("slack_secret.json")
	if err != nil {
		log.Fatalf("Unable to read Slack secret file: %v", err)
	}

	conf := global.slack.conf
	conf, err = slackConfigFromJSON(b, "chat:write:bot", "incoming-webhook", "bot")
	if err != nil {
		log.Fatal("fail to parse client secret:", err)
	}
	global.slack.cacheFile, err = tokenCacheFile("authsrv-slack.json")
	if err != nil {
		log.Fatal("fail to create cache file:", err)
	}
	global.slack.token, err = slackTokenFromFile(global.slack.cacheFile)
	if err != nil {
		log.Println("fail to load token from cache")
		global.slack.token = nil
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Redirect user to consent page to ask for permission
		// for the scopes specified above.
		authURL := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
		err := tmpls.ExecuteTemplate(w, "index.html", authURL)
		if err != nil {
			log.Println(err)
		}
		log.Println(r.Method, r.URL.Path)
	})

	http.HandleFunc("/auth/slack/callback", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		code := r.URL.Query().Get("code")
		tok, err := conf.Exchange(ctx, code)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%#v", tok)
		stok := newSlackToken(tok)
		log.Printf("%#v", stok)
		stok.Save(global.slack.cacheFile)

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("access token for Slack are saved"))
		log.Println(r.Method, r.URL.Path)
	})

	params := url.Values{
		"channel": {"#general"},
		"text":    {"Hello! (using OAut2)"},
	}
	http.HandleFunc("/slack/hello", makeSlakeHandler("https://slack.com/api/chat.postMessage", params))
	http.HandleFunc("/slack/auth.test", makeSlakeHandler("https://slack.com/api/auth.test", nil))
	http.HandleFunc("/slack/bots.info", makeSlakeHandler("https://slack.com/api/bots.info", nil))

	log.Println("listening on :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"net/url"

	"html/template"
	"io/ioutil"

	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/slack"
)

var tmpls = template.Must(template.ParseGlob("*.html"))

var conf struct {
	slack   *oauth2.Config
	authURL string
}

func main() {
	conf.slack = &oauth2.Config{
		ClientID:     os.Getenv("SLACK_KEY"),
		ClientSecret: os.Getenv("SLACK_SECRET"),
		Scopes:       []string{"chat:write:bot"},
		Endpoint:     slack.Endpoint,
	}
	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	conf.authURL = conf.slack.AuthCodeURL("state", oauth2.AccessTypeOffline)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		err := tmpls.ExecuteTemplate(w, "index.html", conf.authURL)
		if err != nil {
			log.Println(err)
		}
		log.Println(r.Method, r.URL.RawPath)
	})
	http.HandleFunc("/auth/slack/callback", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		code := r.URL.Query().Get("code")
		tok, err := conf.slack.Exchange(ctx, code)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(code, tok)

		client := conf.slack.Client(ctx, tok)

		_ = url.Values{
			"channel": {"#general"},
			"text":    {"Hello! (using OAut2)"},
		}
		resp, err := client.Get("https://slack.com/api/chat.postMessage")
		if err != nil {
			log.Fatal(err)
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
	})
	log.Println("listening on :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

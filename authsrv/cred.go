package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/slack"
)

type botToken struct {
	UserID      string `json:"bot_user_id"`
	AccessToken string `json:"bot_access_token"`
}
type webhookToken struct {
	URL       string `json:"url"`
	Chan      string `json:"channel"`
	ConfigURL string `json:"configuration_url"`
}
type slackToken struct {
	*oauth2.Token
	UserID   string        `json:"user_id"`
	TeamName string        `json:"team_name"`
	Bot      *botToken     `json:"bot,omitempty"`
	Webhook  *webhookToken `json:"incoming_webhook,omitempty"`
}

func newSlackToken(tok *oauth2.Token) *slackToken {
	stok := &slackToken{Token: tok}

	iface := tok.Extra("user_id")
	stok.UserID = iface.(string)

	iface = tok.Extra("team_name")
	stok.TeamName = iface.(string)

	iface = tok.Extra("bot")
	fields, ok := iface.(map[string]interface{})
	if ok {
		bot := &botToken{}
		bot.UserID = fields["bot_user_id"].(string)
		bot.AccessToken = fields["bot_access_token"].(string)
		stok.Bot = bot
	}
	iface = tok.Extra("incoming_webhook")
	fields, ok = iface.(map[string]interface{})
	if ok {
		webhook := &webhookToken{}
		webhook.URL = fields["url"].(string)
		webhook.Chan = fields["channel"].(string)
		webhook.ConfigURL = fields["configuration_url"].(string)
		stok.Webhook = webhook
	}
	return stok
}

func slackTokenFromFile(filename string) (*slackToken, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	t := &slackToken{}
	err = json.NewDecoder(f).Decode(t)
	f.Close()
	return t, err
}

func (tok *slackToken) Save(filename string) {
	fmt.Printf("Saving credential file to: %s\n", filename)
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	json.NewEncoder(f).Encode(tok)
	f.Close()
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile(filename string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir, filename), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(filename string) (*oauth2.Token, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(filename string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", filename)
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	json.NewEncoder(f).Encode(token)
	f.Close()
}

// SlackConfigFromJSON load Slack config from a JSON document as followed:
// {"client_id":"myID","client_secret":"mySecret","redirect_uris":["myURI"]}
func slackConfigFromJSON(jsonKey []byte, scope ...string) (*oauth2.Config, error) {
	type cred struct {
		ClientID     string   `json:"client_id"`
		ClientSecret string   `json:"client_secret"`
		RedirectURIs []string `json:"redirect_uris"`
		AuthURI      string   `json:"auth_uri"`
		TokenURI     string   `json:"token_uri"`
	}
	var c cred
	if err := json.Unmarshal(jsonKey, &c); err != nil {
		return nil, err
	}
	if len(c.RedirectURIs) < 1 {
		return nil, errors.New("authsrv: missing redirect URL in the client_credentials.json")
	}
	return &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  c.RedirectURIs[0],
		Scopes:       scope,
		Endpoint:     slack.Endpoint,
	}, nil
}

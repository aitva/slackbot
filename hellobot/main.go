package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type message struct {
	Text string `json:"text"`
}

func main() {
	token := os.Getenv("TOKEN")
	if token == "" {
		fmt.Println("Variable TOKEN must be defined.")
		os.Exit(1)
	}
	url := fmt.Sprintf("https://hooks.slack.com/services/%s", token)

	fmt.Println("I'm a Slack bot and I'm going to say hello.")

	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	err := e.Encode(&message{Text: "Hello Bro!"})
	if err != nil {
		fmt.Println("I've fail to encode message:", err)
		os.Exit(1)
	}

	resp, err := http.Post(url, "application/json", &buf)
	if err != nil {
		fmt.Println("I've fail to communicate with Slack:", err)
		os.Exit(1)
	}

	fmt.Println("resp.StatusCode:", resp.StatusCode)
}

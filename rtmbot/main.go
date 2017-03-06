package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
)

type rtmResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
	URL   string `json:"url"`
	Self  struct {
		ID string `json:"id"`
	} `json:"self"`
}

type rtmMsg struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func fatal(isOK bool, a ...interface{}) {
	if !isOK {
		return
	}
	fmt.Println(a...)
	os.Exit(1)
}

func main() {
	token := os.Getenv("TOKEN")
	fatal(token == "", "Variable TOKEN must be defined.")

	fmt.Println("Starting RTM service...")
	url := fmt.Sprintf("https://slack.com/api/rtm.start?token=%s", token)

	resp, err := http.Get(url)
	fatal(err != nil, "fail to reach server:", err)
	fatal(resp.StatusCode != http.StatusOK, "unexpected status code:", resp.StatusCode)

	var rtm rtmResponse
	err = json.NewDecoder(resp.Body).Decode(&rtm)
	fatal(err != nil, "fail to parse response:", err)

	fmt.Println("Connecting to RTM service...")
	c, _, err := websocket.DefaultDialer.Dial(rtm.URL, nil)
	fatal(err != nil, "connection fail:", err)
	defer c.Close()

	channels := make(chan rtmMsg)
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				fmt.Fprintln(os.Stderr, "fail to read message:", err)
				return
			}
			fmt.Println(string(message))

			msg := rtmMsg{}
			err = json.Unmarshal(message, &msg)
			if err != nil {
				fmt.Fprintln(os.Stderr, "fail to parse message:", err)
				return
			}
			if msg.Type != "message" {
				continue
			}
			channels <- msg
		}
	}()

	go func() {
		id := 0
		for {
			req := <-channels
			resp := rtmMsg{
				ID:      id,
				Type:    "message",
				Text:    "Hello!",
				Channel: req.Channel,
			}

			err := c.WriteJSON(&resp)
			if err != nil {
				fmt.Println("fail to send message:", err)
				return
			}
			id++
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	fmt.Println("Closing RTM connection...")
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	err = c.WriteMessage(websocket.CloseMessage, msg)
	fatal(err != nil, "fail to close socker:", err)
}

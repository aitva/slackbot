package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"strings"

	"github.com/gorilla/websocket"
)

var global struct {
	StartMsg rtmResponse
}

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

func readRTM(c *websocket.Conn, channels chan<- rtmMsg) {
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
}

func writeRTM(c *websocket.Conn, channels <-chan rtmMsg) {
	id := 0
	botname := "<@" + global.StartMsg.Self.ID + ">"
	for {
		req := <-channels
		if !strings.HasPrefix(req.Text, botname) {
			continue
		}
		i := strings.Index(req.Text, ":")
		trimed := strings.Trim(req.Text[i+1:], " ")
		fmt.Fprintln(os.Stderr, "i:", i, "trimed:", trimed)
		all := strings.Split(trimed, " ")
		if len(all) == 0 || len(all) > 2 {
			fmt.Fprintln(os.Stderr, "fail to parse command:", req.Text)
			continue
		}

		resp := rtmMsg{
			ID:      id,
			Type:    "message",
			Channel: req.Channel,
		}
		switch all[0] {
		case "hello":
			resp.Text = "Hello!"
		case "bye":
			resp.Text = "Bye!"
		default:
			fmt.Fprintln(os.Stderr, "unexpected command:", all)
			continue
		}

		err := c.WriteJSON(&resp)
		if err != nil {
			fmt.Fprintln(os.Stderr, "fail to send message:", err)
			return
		}
		id++
	}
}

func main() {
	token := os.Getenv("TOKEN")
	fatal(token == "", "Variable TOKEN must be defined.")

	fmt.Println("Starting RTM service...")
	url := fmt.Sprintf("https://slack.com/api/rtm.start?token=%s", token)

	resp, err := http.Get(url)
	fatal(err != nil, "fail to reach server:", err)
	fatal(resp.StatusCode != http.StatusOK, "unexpected status code:", resp.StatusCode)

	err = json.NewDecoder(resp.Body).Decode(&global.StartMsg)
	fatal(err != nil, "fail to parse response:", err)

	fmt.Println("Connecting to RTM service...")
	c, _, err := websocket.DefaultDialer.Dial(global.StartMsg.URL, nil)
	fatal(err != nil, "connection fail:", err)
	defer c.Close()

	channels := make(chan rtmMsg)
	go readRTM(c, channels)
	go writeRTM(c, channels)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	fmt.Println("Closing RTM connection...")
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	err = c.WriteMessage(websocket.CloseMessage, msg)
	fatal(err != nil, "fail to close socker:", err)
}

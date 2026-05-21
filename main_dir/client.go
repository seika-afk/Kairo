package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	session *Session
	conn    *websocket.Conn
	send    chan []byte
}

func (c *Client) readPump() {
	defer func() {
		c.session.Unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.session.Broadcast <- message

	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		select {

		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		}
	}

}
func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	var join Message
	err = conn.ReadJSON(&join)
	if err != nil {
		return
	}
	if join.Kind == "join" {
		fmt.Println("Client Joined With session id : ", join.SessionID)
	}
	if join.Kind != "join" {
		return
	}
	session := sm.getSession(join.SessionID)
	if join.Doc != nil {
		session.Doc = []rune(*join.Doc)
	}

	client := &Client{session: session, conn: conn, send: make(chan []byte, 256)}
	client.session.Register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()

	joinBytes, err := json.Marshal(join)
	if err != nil {
		return
	}
	session.Mu.Lock()
	for otherClient := range session.Clients {
		if otherClient == client {
			continue
		}
		select {
		case otherClient.send <- joinBytes:
		default:
			close(otherClient.send)
			delete(session.Clients, otherClient)
		}
	}
	session.Mu.Unlock()

	initPayload := InitPayload{
		Kind: "init",
		Doc:  string(session.Doc),
	}

	initBytes, err := json.Marshal(initPayload)
	if err != nil {
		return
	}
	client.send <- initBytes
}

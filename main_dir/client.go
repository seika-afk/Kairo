package main

import (
	"encoding/json"
	"kairo/ot"
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
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:3000" || origin == ""
	},
}

type Client struct {
	session *Session
	id      string
	conn    *websocket.Conn
	send    chan []byte
}
type ClientOp struct {
	Client *Client
	Op     ot.Op
}
type CursorEnvelope struct {
	Sender *Client
	Data   []byte
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
		var cursor Cursor
		if err := json.Unmarshal(message, &cursor); err == nil && cursor.Kind == "cursor" {
			c.session.CursorBroadcast <- CursorEnvelope{
				Sender: c,
				Data:   message,
			}
			continue
		}

		var op ot.Op
		err = json.Unmarshal(message, &op)
		if err != nil {
			log.Printf("ignoring malformed op ")
			continue
		}

		if op.Type != "insert" && op.Type != "delete" {
			log.Printf("ignoring non-op websocket message from ")
			continue
		}
		c.session.IncomingOp <- ClientOp{
			Client: c,
			Op:     op,
		}

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
		log.Printf("failed to read join message from %v", err)
		return
	}
	if join.Kind == "join" {
		log.Printf("client joined session=%s client_id=%s ", join.SessionID, join.ClientID)
	}
	if join.Kind != "join" {
		log.Printf("ignoring first websocket message kind=%q", join.Kind)
		return
	}
	session := sm.getSession(join.SessionID)
	if join.Doc != nil {
		session.Doc = []rune(*join.Doc)
	}

	client := &Client{session: session, id: join.ClientID, conn: conn, send: make(chan []byte, 256)}
	client.session.Register <- client

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

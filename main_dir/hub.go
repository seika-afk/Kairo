package main

import (
	"encoding/json"
	"sync"

	"kairo/ot"
)

// can also call this file session.go : uh cuz it has sessions but if we talk in respect to websocket its also like hub

type Session struct {
	//session specfic
	ID  string
	Doc []rune
	//hub specific
	Clients       map[*Client]bool
	RemoteCursors map[string]Cursor
	Broadcast     chan []byte

	Register        chan *Client
	Unregister      chan *Client
	IncomingOp      chan ClientOp
	CursorBroadcast chan CursorEnvelope

	Version int
	History []ot.Op

	Mu sync.Mutex
}

type SessionManager struct {
	Sessions map[string]*Session
	Mu       sync.Mutex
}

func newSession(id string) *Session {
	return &Session{
		ID:  id,
		Doc: []rune{},

		Broadcast:       make(chan []byte),
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
		IncomingOp:      make(chan ClientOp),
		CursorBroadcast: make(chan CursorEnvelope),

		Clients:       make(map[*Client]bool),
		RemoteCursors: make(map[string]Cursor),

		Version: 0,
		History: []ot.Op{},
	}
}

func (m *SessionManager) getSession(id string) *Session {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	session, exists := m.Sessions[id]
	if exists {
		return session
	}
	session = newSession(id)
	m.Sessions[id] = session
	go session.run()
	return session

}
func (h *Session) run() {
	for {
		select {
		case client := <-h.Register:
			h.Mu.Lock()
			h.Clients[client] = true
			h.Mu.Unlock()
		case client := <-h.Unregister:
			h.Mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.send)
				delete(h.RemoteCursors, client.id)
				clearCursorBytes, err := json.Marshal(Cursor{
					Kind:     "cursor_clear",
					ClientID: client.id,
				})
				if err == nil {
					for otherClient := range h.Clients {
						select {
						case otherClient.send <- clearCursorBytes:
						default:
							close(otherClient.send)
							delete(h.Clients, otherClient)
						}
					}
				}
			}
			h.Mu.Unlock()
		case message := <-h.Broadcast:
			h.Mu.Lock()
			for client := range h.Clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.Clients, client)
				}
			}
			h.Mu.Unlock()
		case incoming := <-h.IncomingOp:
			h.Mu.Lock()
			transformed := ot.TransformAgainstHistory(incoming.Op, incoming.Client.session.History, incoming.Op.Version)
			incoming.Client.session.Doc = ot.Apply(incoming.Client.session.Doc, transformed)

			//inc version and add in history
			incoming.Client.session.Version++
			transformed.Version = incoming.Client.session.Version

			incoming.Client.session.History = append(incoming.Client.session.History, transformed)

			// apply the transformed op to all other clients in the session
			opBytes, err := json.Marshal(transformed)
			if err == nil {
				for otherClient := range h.Clients {
					if otherClient == incoming.Client {
						continue
					}
					select {
					case otherClient.send <- opBytes:
					default:
						close(otherClient.send)
						delete(h.Clients, otherClient)
					}
				}
			}

			h.Mu.Unlock()

		case cursorEnvelope := <-h.CursorBroadcast:
			h.Mu.Lock()
			var cursor Cursor
			if err := json.Unmarshal(cursorEnvelope.Data, &cursor); err == nil {
				// The websocket sender owns the cursor identity; do not trust the payload's client_id.
				cursor.ClientID = cursorEnvelope.Sender.id
				normalizedCursorBytes, marshalErr := json.Marshal(cursor)
				if marshalErr == nil {
					cursorEnvelope.Data = normalizedCursorBytes
				}
				h.RemoteCursors[cursor.ClientID] = cursor
			}
			for client := range h.Clients {

				// don't send back to sender
				if client == cursorEnvelope.Sender {
					continue
				}

				select {

				case client.send <- cursorEnvelope.Data:

				default:
					close(client.send)
					delete(h.Clients, client)
				}
			}
			h.Mu.Unlock()
		}
	}
}

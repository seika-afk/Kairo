package main

import "sync"

// can also call this file session.go : uh cuz it has sessions but if we talk in respect to websocket its also like hub

type Session struct {
	//session specfic
	ID  string
	Doc []rune
	//hub specific
	Clients   map[*Client]bool
	Broadcast chan []byte

	Register   chan *Client
	Unregister chan *Client

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

		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
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
		}
	}
}

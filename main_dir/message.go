package main

type Message struct {
	Kind      string  `json:"kind"`
	SessionID string  `json:"session_id"`
	ClientID  string  `json:"client_id,omitempty"`
	Doc       *string `json:"doc,omitempty"`
}

type InitPayload struct {
	Kind string `json:"kind"`
	Doc  string `json:"doc"`
}

type Cursor struct {
	Kind     string `json:"kind"`
	ClientID string `json:"client_id"`
	Line     int    `json:"line"`
	Position int    `json:"position"`
}

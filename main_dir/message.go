package main

type Message struct {
	Kind      string  `json:"kind"`
	SessionID string  `json:"session_id"`
	Doc       *string `json:"doc,omitempty"`
}

type InitPayload struct {
	Kind string `json:"kind"`
	Doc  string `json:"doc"`
}

package main

type Message struct {
	Kind      string `json:"kind"`
	SessionID string `json:"session_id"`
}

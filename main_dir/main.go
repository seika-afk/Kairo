package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Session struct {
	Id      string
	Doc     []rune
	Clients map[string]*Client
	Mu      sync.Mutex
}

type SessionManager struct {
	Sessions map[string]*Session
	Mu       sync.Mutex
}

var addr = flag.String("addr", ":4000", "http service address")

func main() {
	flag.Parse()
	hub := newHub()
	go hub.run()
	fmt.Println("WS Server started at : ws://localhost:", *addr, "/ws")
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	err := http.ListenAndServe(*addr, nil)

	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

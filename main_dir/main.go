package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":4000", "http service address")
var sm = SessionManager{
	Sessions: make(map[string]*Session),
}

func main() {
	flag.Parse()

	fmt.Println("WS Server started at : ws://localhost:", *addr, "/ws")
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(w, r)
	})
	err := http.ListenAndServe(*addr, nil)

	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

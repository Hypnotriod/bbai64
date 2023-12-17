package main

import (
	"bbai64/command"
	"bbai64/vehicle"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

const SERVER_ADDRESS = ":1337"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin:     checkOrigin,
}

var wsMutex sync.Mutex

func checkOrigin(r *http.Request) bool {
	return true
}

func serveWSRequest(w http.ResponseWriter, r *http.Request) {
	if !wsMutex.TryLock() {
		log.Print("Websocket multiple connections are not allowed with ", r.Host)
		return
	}
	defer wsMutex.Unlock()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Websocket upgrade error: ", err)
		return
	}
	log.Print("Websocket connection established with ", r.Host)
	defer conn.Close()
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Print("Websocket read error: ", err)
			break
		}
		cmd, err := command.Unmarshal(message)
		if err != nil {
			log.Print("Websocket command format error: ", err)
			break
		}
		vehicle.ProcessCommand(cmd)
	}
	log.Print("Websocket connection terminated with ", r.Host)
}

func main() {
	vehicle.Initialize()

	http.HandleFunc("/ws", serveWSRequest)
	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.ListenAndServe(SERVER_ADDRESS, nil)
}

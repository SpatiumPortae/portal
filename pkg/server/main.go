package server

import (
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var upgrader = websocket.Upgrader{}

func main() {
	// set up a goroutine listening for new send requests and pushing them to a queue-stack
	// set up a goroutine listening for new receive requests and find the desired connection in the queue stack

	flag.Parse()

	http.HandleFunc("/establish-send", establishSendEndpoint)
	// http.HandleFunc("/send", receiveEndpoint)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func establishSendEndpoint(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("failed to upgrade connection: ", err)
		return
	}
	defer c.Close()

	// read initial send request from sender
	f := models.File{}
	err = c.ReadJSON(&f)
	if err != nil {
		log.Println("failed to read initial send request message: ", err)
		return
	}
	

}
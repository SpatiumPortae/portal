package server

import (
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var upgrader = websocket.Upgrader{}

func main() {
	// set up a goroutine listening for new send requests and pushing them to a queue-stack
	// set up a goroutine listening for new receive requests and find the desired connection in the queue stack

	flag.Parse()

	http.HandleFunc("/receive", receiveEndpoint)
	// http.HandleFunc("/send", receiveEndpoint)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func receiveEndpoint(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("failed to upgrade connection: ", err)
		return
	}
	defer c.Close()


	err := c.ReadJSON()
	if err != nil {
		log.Println("failed to read initial receive request message: ", err)
		return
	}

}
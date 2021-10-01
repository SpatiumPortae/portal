package server

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var upgrader = websocket.Upgrader{}

var server = http.Server{
	Addr: ":8080",
	ReadTimeout: 30 * time.Second,
	WriteTimeout: 30 * time.Second,
}

func start() {
	// set up a goroutine listening for new send requests and pushing them to a queue-stack
	// set up a goroutine listening for new receive requests and find the desired connection in the queue stack

	flag.Parse()

	http.HandleFunc("/establish-send", establishSendEndpoint)
	// http.HandleFunc("/send", receiveEndpoint)
	log.Fatal(http.ListenAndServe(*addr, nil))
}


func establishReceiveEndpoint(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
}

func establishSendEndpoint(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("failed to upgrade connection: ", err)
		return
	}
	defer c.Close()

	mailbox := Mailbox{
		Sender: NewClient(c), 
	}

	// read initial send request from sender
	f := models.File{}
	err = c.ReadJSON(&f)
	if err != nil {
		log.Println("failed to read initial send request message: ", err)
		return
	}

	// skapa mailbox och stoppa in File och Sender (men utan port)
	// mailbox.Sender.
}
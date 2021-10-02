package server

import (
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	router    *http.ServeMux
	mailboxes *Mailboxes
}

var server Server
var httpServer *http.Server

func init() {
	rand.Seed(time.Now().UnixNano())
	
	server = Server{
		router:    http.NewServeMux(),
		mailboxes: &Mailboxes{&sync.Map{}},
	}

	httpServer = &http.Server{
		Addr:         ":6969",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      server.router,
	}

	server.routes()
}

func Start() {
	log.Fatal(httpServer.ListenAndServe())
}

package server

import (
	"log"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	httpServer *http.Server
	router     *http.ServeMux
	mailboxes  *Mailboxes
}

var server Server

func init() {
	server = Server{
		httpServer: &http.Server{
			Addr:         ":6969",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			Handler:      server.router,
		},
		router:    http.NewServeMux(),
		mailboxes: &Mailboxes{&sync.Map{}},
	}

	server.routes()
}

func Start() {
	log.Fatal(server.httpServer.ListenAndServe())
}

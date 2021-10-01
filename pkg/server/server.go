package server

import (
	"log"
	"net/http"
	"time"
)

type Server struct {
	router *http.ServeMux
	mailboxes *Mailboxes
}

var server Server
var httpServer *http.Server

func init() {
	server = Server{
		router: http.NewServeMux(),
		mailboxes: &Mailboxes{},
	}
	
	httpServer = &http.Server{
		Addr: 						 ":8080",
		ReadTimeout:       30 * time.Second,
    WriteTimeout:      30 * time.Second,
		Handler: server.router,
	}
}

func start() {
	log.Fatal(httpServer.ListenAndServe())
}

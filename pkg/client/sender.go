package client

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/tools"
)

// Server is small webserver for transfer the a file once.
type Server struct {
	server  *http.Server
	router  *http.ServeMux
	payload []byte
}

// NewServer creates a new client.Server struct.
func NewServer(port int64, payload []byte) (*Server, error) {
	router := &http.ServeMux{}
	s := &Server{
		router: router,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port), //TODO: set IP as well as port
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			Handler:      router,
		},
		payload: payload,
	}
	s.routes()
	return s, nil
}

func (s *Server) Start() {
	log.Fatal(s.server.ListenAndServe())
}

func (s *Server) routes() {
	s.router.HandleFunc("/portal", s.handleTransfer())
	s.router.HandleFunc("/ping", s.handlePing())
}

// handleTransfer creates a HandlerFunc to handle the transfer of files.
func (s *Server) handleTransfer() http.HandlerFunc {
	transferHandleFunc := func(wsConn *websocket.Conn) {
		err := wsConn.WriteMessage(websocket.TextMessage, s.payload) //TODO: some abstraction for file/dir/message
		if err != nil {
			log.Println("Could not send payload")
		}
		wsConn.Close()
		s.server.Shutdown(context.Background()) //TODO: dont close silent?.
	}
	return tools.WebsocketHandler(transferHandleFunc)
}

func (s *Server) handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Pong")
	}
}

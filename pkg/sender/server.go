package sender

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/tools"
)

// Server is small webserver for transfer the a file once.
type Server struct {
	server       *http.Server
	router       *http.ServeMux
	upgrader     websocket.Upgrader
	payload      []byte
	receiverAddr net.Addr
}

// NewServer creates a new client.Server struct.
func NewServer(port int64, payload []byte, recevierAddr net.Addr) (*Server, error) {
	router := &http.ServeMux{}
	s := &Server{
		router: router,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port), //TODO: set IP as well as port
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			Handler:      router,
		},
		upgrader:     websocket.Upgrader{},
		payload:      payload,
		receiverAddr: recevierAddr,
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
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if the client has correct address.
		if r.RemoteAddr != s.receiverAddr.String() {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "No Portal for You!")
			log.Println("Portal attempt from alien spieces...")
			return
		}
		// Establish websocket connection
		wsConn, err := s.upgrader.Upgrade(w, r, nil)
		err = wsConn.WriteMessage(websocket.BinaryMessage, s.payload) //TODO: some abstraction for file/dir/message
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Technical difficulties with the Portal.")
			log.Println("Could not send data through Portal.")
			return
		}
		//TODO: Gracefully sutdown server
		s.server.Shutdown(context.Background())
		return
	}
}

func (s *Server) handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Pong")
	}
}

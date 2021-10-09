package sender

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/protocol"
)

// Server is small webserver for transfer the a file once.
type Server struct {
	server       *http.Server
	router       *http.ServeMux
	upgrader     websocket.Upgrader
	payload      []byte
	receiverAddr net.IP
}

// NewServer creates a new client.Server struct.
func NewServer(port int64, payload []byte, recevierAddr net.IP) (*Server, error) {
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
		if s.receiverAddr.Equal(net.ParseIP(r.RemoteAddr)) {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "No Portal for You!")
			log.Printf("Portal attempt from alien spieces with IP:%q...", r.RemoteAddr)
			return
		}

		// Establish websocket connection
		wsConn, err := s.upgrader.Upgrade(w, r, nil)

		// Wait for client to annouce its readyness
		clientReady := &protocol.TransferClientReadyMessage{}
		err = wsConn.ReadJSON(clientReady)
		if err != nil {
			log.Printf("Alien error: %q", err)
			wsConn.WriteJSON(protocol.TransferServerErrorMessage{Error: err})
		}

		if !clientReady.Ready {
			// Handle this
		}

		// annouce that server is ready
		err = wsConn.WriteJSON(protocol.TransferServerReadyMessage{Ready: true})
		if err != nil {
			log.Fatalf("Alien error: %q", err)
		}

		// Send payload to the client
		err = wsConn.WriteMessage(websocket.BinaryMessage, s.payload)
		if err != nil {
			wsConn.WriteJSON(protocol.TransferServerErrorMessage{Error: err})
		}

		clientClose := &protocol.TransferClientClosingMessage{}
		err = wsConn.ReadJSON(clientClose)
		if err != nil {
			log.Println(err)
		}

		wsConn.WriteJSON(protocol.TransferServerClosingMessage{Closing: true})

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

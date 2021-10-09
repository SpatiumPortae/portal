package sender

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/portal"
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
		if err != nil {
			log.Printf("Unable to intililize Portal due to technical error: %q\n", err)
			//TODO: sutdown gracefully
		}
		defer wsConn.Close()

		for {
			_, b, err := wsConn.ReadMessage()

			if err != nil {
				log.Println(err)
			}
			msg := &portal.TransferMessage{}
			err = json.Unmarshal(b, msg)
			if err != nil {
				log.Println(err)
			}

			switch msg.Type {
			case portal.ClientHandshake:
				wsConn.WriteJSON(portal.TransferMessage{
					Type:    portal.ServerHandshake,
					Message: "Portal intiliazied.",
				})
			case portal.ClientRequestPayload:
				// TODO: handle multiple payloads?
				// Send payload
				wsConn.WriteMessage(websocket.BinaryMessage, s.payload)
			case portal.ClientAckPayload:
				// handle multiple payloads.
			case portal.Error:
				log.Printf("Shutting down portal due to Alien error: %q\n", msg.Message)
				//TODO: Shutdown gracefully
				return
			case portal.ClientClosing:
				wsConn.WriteJSON(portal.TransferMessage{
					Type:    portal.ServerClosing,
					Message: "Closing down the Portal, as requested.",
				})
				//TODO: Sutdown gracefully
				return
			}
		}
	}
}

func (s *Server) handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Pong")
	}
}

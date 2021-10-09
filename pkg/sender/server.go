package sender

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/portal"
)

// Server is small webserver for transfer the a file once.
type Server struct {
	server       *http.Server
	router       *http.ServeMux
	upgrader     websocket.Upgrader
	payload      []byte //NOTE: Handle multiple payloads?
	receiverAddr net.IP
	done         chan os.Signal
}

// NewServer creates a new client.Server struct.
func NewServer(port int64, payload []byte, recevierAddr net.IP) (*Server, error) {
	router := &http.ServeMux{}
	s := &Server{
		router: router,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			Handler:      router,
		},
		upgrader:     websocket.Upgrader{},
		payload:      payload,
		receiverAddr: recevierAddr,
		done:         make(chan os.Signal, 1),
	}
	// hook up os signals to the done chanel.
	signal.Notify(s.done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	s.routes()
	return s, nil
}

// Start starts the sender.Server webserver.
func (s *Server) Start() {
	// context used for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())

	// Start shutdown sequence.
	go func() {
		osCall := <-s.done //listen for OS signals.
		log.Printf("Initializing Portal shutdown sequence, system call: %s\n", osCall)
		cancel() // cancel the context.
	}()

	// serve the webserver, and report errors.
	if err := serve(s, ctx); err != nil {
		log.Printf("Unable to serve Portal, due to technical error: %s\n", err)
	}
}

// serve is helper function that serves the webserver while providing graceful shutdown.
func serve(s *Server, ctx context.Context) (err error) {
	go func() {
		if err = s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Serving Portal: %s\n", err)
		}
	}()

	log.Println("Portal Server has started.")
	<-ctx.Done() // wait for the shutdown sequence to start.

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	// sutdown and report errors.
	if err = s.server.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Portal shutdown sequence failed to due error:%s", err)
	}

	// strip error in this case, as we deal with this gracefully.
	if err == http.ErrServerClosed {
		err = nil
	}
	log.Println("Portal shutdown successfully.")
	return err
}

// routes is a helper function used for setting up the routes.
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
			log.Printf("Unauthorized Portal attempt from alien species with IP: %s.\n", r.RemoteAddr)
			return
		}

		// Establish websocket connection.
		wsConn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Unable to intilize Portal due to technical error: %s.\n", err)
			s.done <- syscall.SIGTERM
			return
		}
		log.Printf("Established Potal connection with alien species with IP: %s.\n", r.RemoteAddr)
		defer wsConn.Close()

		for {
			msg := &portal.TransferMessage{}
			err := wsConn.ReadJSON(msg)

			if err != nil {
				log.Println(err)
			}

			switch msg.Type {
			case portal.ClientHandshake:
				wsConn.WriteJSON(portal.TransferMessage{
					Type:    portal.ServerHandshake,
					Message: "Portal initialized.",
				})
			case portal.ClientRequestPayload:
				// TODO: handle multiple payloads?
				// Send payload.
				wsConn.WriteMessage(websocket.BinaryMessage, s.payload)
			case portal.ClientAckPayload:
				// handle multiple payloads.
			case portal.Error:
				log.Printf("Shutting down Portal due to Alien error: %q\n", msg.Message)
				s.done <- syscall.SIGTERM
				return
			case portal.ClientClosing:
				wsConn.WriteJSON(portal.TransferMessage{
					Type:    portal.ServerClosing,
					Message: "Closing down the Portal, as requested.",
				})
				s.done <- syscall.SIGTERM
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

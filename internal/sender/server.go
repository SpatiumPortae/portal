// server.go defines the sender webserver for the Portal file transfer
package sender

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/internal/conn"
)

// Server specifies the webserver that will be used for direct file transfer.
type Server struct {
	server   *http.Server
	router   *http.ServeMux
	upgrader websocket.Upgrader

	Err      error
	shutdown chan os.Signal
	once     sync.Once
}

// NewServer creates a new server running on the provided port.
func NewServer(port int, key []byte, payload io.Reader, payloadSize int64, msgs ...chan interface{}) *Server {
	router := &http.ServeMux{}
	s := &Server{
		router: router,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			Handler:      router,
		},
		upgrader: websocket.Upgrader{},
	}
	s.shutdown = make(chan os.Signal)
	signal.Notify(s.shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	// setup routes
	router.HandleFunc("/portal", s.handleTransfer(key, payload, payloadSize, msgs...))
	return s
}

// Start starts the server and sets up graceful shutdown.
func (s *Server) Start() error {
	ctx := context.Background()
	idleConnsClosed := make(chan struct{})
	go func() {
		<-s.shutdown
		log.Println("received cancel signal")
		if err := s.server.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	<-idleConnsClosed
	return nil
}

// Shutdown shutdowns the server. Is safe to call multiple times as ONLY 1
// shutdown signal will ever be generated.
func (s *Server) Shutdown() {
	s.once.Do(func() {
		s.shutdown <- syscall.SIGTERM
	})
}

// handleTransfer returns a HTTP handler that preforms the transfer sequence.
// Will shutdown the server on termination.
func (s *Server) handleTransfer(key []byte, payload io.Reader, payloadSize int64, msgs ...chan interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			s.Shutdown()
		}()
		ws, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.Err = err
			return
		}
		tc := conn.TransferFromKey(&conn.WS{Conn: ws}, key)
		if err != transfer(tc, payload, payloadSize, msgs...) {
			s.Err = err
			return
		}
	}
}

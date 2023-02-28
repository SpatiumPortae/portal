// server.go defines the sender webserver for the Portal file transfer
package sender

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/SpatiumPortae/portal/internal/conn"
	"nhooyr.io/websocket"
)

// server specifies the webserver that will be used for direct file transfer.
type server struct {
	server *http.Server
	router *http.ServeMux

	Err      error
	shutdown chan os.Signal
	once     sync.Once
}

// newServer creates a new server running on the provided port.
func newServer(port int, key []byte, payload io.Reader, payloadSize int64, msgs ...chan interface{}) *server {
	router := &http.ServeMux{}
	s := &server{
		router: router,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			Handler:      router,
		},
	}
	s.shutdown = make(chan os.Signal)
	signal.Notify(s.shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	// setup routes
	router.HandleFunc("/portal", s.handleTransfer(key, payload, payloadSize, msgs...))
	return s
}

// Start starts the server and sets up graceful shutdown.
func (s *server) Start() error {
	idleConnsClosed := make(chan struct{})
	var shutdownErr error
	go func() {
		<-s.shutdown
		if err := s.server.Shutdown(context.Background()); err != nil {
			shutdownErr = err
		}
		close(idleConnsClosed)
	}()
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	<-idleConnsClosed
	return shutdownErr
}

// Shutdown shutdowns the server. Is safe to call multiple times as ONLY 1
// shutdown signal will ever be generated.
func (s *server) Shutdown() {
	s.once.Do(func() {
		s.shutdown <- syscall.SIGTERM
	})
}

// handleTransfer returns a HTTP handler that performs the transfer sequence.
// Will shutdown the server on termination.
func (s *server) handleTransfer(key []byte, payload io.Reader, payloadSize int64, msgs ...chan interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			s.Shutdown()
		}()
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			s.Err = err
			return
		}
		tc := conn.TransferFromKey(&conn.WS{Conn: ws}, key)
		if err != transferSequence(context.Background(), tc, payload, payloadSize, msgs...) {
			s.Err = err
			return
		}
	}
}

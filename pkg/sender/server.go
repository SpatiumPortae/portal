// server.go defines the sender webserver for the Portal file transfer
package sender

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// Server is small webserver for transfer the a file once.
type Server struct {
	server       *http.Server
	router       *http.ServeMux
	upgrader     websocket.Upgrader
	payload      io.Reader
	receiverAddr net.IP
	done         chan os.Signal
	logger       *log.Logger
}

// NewServer creates a new client.Server struct.
func NewServer(port int64, payload io.Reader, recevierAddr net.IP, logger *log.Logger) (*Server, error) {
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
		logger:       logger,
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
		s.logger.Printf("Initializing Portal shutdown sequence, system call: %s\n", osCall)
		cancel() // cancel the context.
	}()

	// serve the webserver, and report errors.
	if err := serve(s, ctx); err != nil {
		s.logger.Printf("Unable to serve Portal, due to technical error: %s\n", err)
	}
}

// serve is helper function that serves the webserver while providing graceful shutdown.
func serve(s *Server, ctx context.Context) (err error) {
	go func() {
		if err = s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatalf("Serving Portal: %s\n", err)
		}
	}()

	s.logger.Println("Portal Server has started.")
	<-ctx.Done() // wait for the shutdown sequence to start.

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	// sutdown and report errors.
	if err = s.server.Shutdown(ctxShutdown); err != nil {
		s.logger.Fatalf("Portal shutdown sequence failed to due error:%s", err)
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

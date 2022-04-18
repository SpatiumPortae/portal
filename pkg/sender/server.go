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
	"www.github.com/ZinoKader/portal/internal/conn"
)

// Server specifies the webserver that will be used for direct file transfer.
type Server struct {
	server   *http.Server
	router   *http.ServeMux
	upgrader websocket.Upgrader
	shutdown chan os.Signal
}

// Specifies the necessary options for initializing the webserver.
type ServerOptions struct {
	port       int
	receiverIP net.IP
}

func NewServer(port int, key []byte, payload io.Reader, writers ...io.Writer) *Server {
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
	router.HandleFunc("/portal", s.handleTransfer(key, payload, writers...))
	return s
}

func (s *Server) Start() error {

	idleConnsClosed := make(chan struct{})
	go func() {
		<-s.shutdown
		if err := s.server.Shutdown(context.Background()); err != nil {
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

func (s *Server) Shutdown() {
	s.shutdown <- syscall.SIGTERM
}

func (s *Server) handleTransfer(key []byte, payload io.Reader, writers ...io.Writer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			// handle error somehow
			return
		}
		tc := conn.TransferFromKey(&conn.WS{Conn: ws}, key)
		if err != transfer(tc, payload, writers...) {
			// handle error somehow
			return
		}
	}
}

// Start starts the sender.Server webserver and setups graceful shutdown
func (s *Sender) StartServer() error {
	if s.senderServer == nil {
		return fmt.Errorf("start called with uninitialized senderServer")
	}
	// context used for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		osCall := <-s.closeServer
		log.Printf("Shutting down Portal sender-server due to system call: %s\n", osCall)
		cancel() // cancel the context
	}()

	// serve the webserver, and report errors
	if err := serve(s, ctx); err != nil {
		return err
	}
	return nil
}

func (s *Sender) CloseServer() {
	s.closeServer <- syscall.SIGTERM
}

// serve is helper function that serves the webserver while providing graceful shutdown.
func serve(s *Sender, ctx context.Context) (err error) {
	if s.senderServer == nil {
		return fmt.Errorf("serve called with uninitialized senderServer")
	}

	go func() {
		if err = s.senderServer.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Portal sender-server crashed due to an error: %s\n", err)
		}
	}()

	log.Println("Portal sender-server started")
	<-ctx.Done() // wait for the shutdown sequence to start.

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	// shutdown and report errors
	if err = s.senderServer.server.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Portal shutdown sequence failed to due error:%s", err)
	}

	// strip error in this case, as we deal with this gracefully
	if err == http.ErrServerClosed {
		err = nil
	}
	log.Println("Portal shutdown successfully")
	return err
}

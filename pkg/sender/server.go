// server.go defines the sender webserver for the Portal file transfer
package sender

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type Server struct {
	server   *http.Server
	router   *http.ServeMux
	upgrader websocket.Upgrader
}

type ServerOptions struct {
	port       int
	receiverIP net.IP
}

// WithServer specifies the option to run the sender by hosting a server which the receiver establishes a connection to
func WithServer(s *Sender, options ServerOptions) *Sender {
	s.receiverIP = options.receiverIP
	router := &http.ServeMux{}
	s.senderServer = &Server{
		router: router,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", options.receiverIP),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			Handler:      router,
		},
		upgrader: websocket.Upgrader{},
	}

	// setup routes
	router.HandleFunc("/portal", s.handleTransfer())
	return s
}

// Start starts the sender.Server webserver and setups graceful shutdown
func (s *Sender) StartServer() error {
	if s.senderServer == nil {
		return fmt.Errorf("start called with uninitialized senderServer\n")
	}
	// context used for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		osCall := <-s.closeServer
		s.logger.Printf("Initializing Portal shutdown sequence, system call: %s\n", osCall)
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
			s.logger.Fatalf("Serving Portal: %s\n", err)
		}
	}()

	s.logger.Println("Portal Server has started.")
	<-ctx.Done() // wait for the shutdown sequence to start.

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	// shutdown and report errors
	if err = s.senderServer.server.Shutdown(ctxShutdown); err != nil {
		s.logger.Fatalf("Portal shutdown sequence failed to due error:%s", err)
	}

	// strip error in this case, as we deal with this gracefully
	if err == http.ErrServerClosed {
		err = nil
	}
	log.Println("Portal shutdown successfully")
	return err
}

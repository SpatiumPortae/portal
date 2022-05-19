package rendezvous

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// Server is contains the necessary data to run the rendezvous server.
type Server struct {
	httpServer *http.Server
	router     *mux.Router
	mailboxes  *Mailboxes
	ids        *IDs
	signal     chan os.Signal
}

// NewServer constructs a new Server struct and setups the routes.
func NewServer(port int) *Server {
	router := &http.ServeMux{}
	s := &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			Handler:      router,
		},
		router:    router,
		mailboxes: &Mailboxes{&sync.Map{}},
		ids:       &IDs{&sync.Map{}},
	}
	s.routes()
	return s
}

// Start runs the rendezvous server.
func (s *Server) Start() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sysCall := <-s.signal
		log.Printf("Portal rendezvous server shutting down due to syscall: %s\n", sysCall)
		cancel()
	}()

	if err := serve(s, ctx); err != nil {
		log.Printf("Error serving Portal rendezvous server: %s\n", err)
	}
}

// serve is a helper function providing graceful shutdown of the server.
func serve(s *Server, ctx context.Context) (err error) {
	go func() {
		if err = s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Serving Portal: %s\n", err)
		}
	}()

	log.Printf("Portal Rendezvous Server started at \"%s\" \n", s.httpServer.Addr)
	<-ctx.Done()

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err = s.httpServer.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Portal rendezvous shutdown failed: %s", err)
	}

	if err == http.ErrServerClosed {
		err = nil
	}
	log.Println("Portal Rendezvous Server shutdown successfully")
	return err
}

package rendezvous

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Server struct {
	httpServer *http.Server
	router     *http.ServeMux
	mailboxes  *Mailboxes
	ids        *IDs
	signal     chan os.Signal
}

func NewServer() *Server {
	router := &http.ServeMux{}
	s := &Server{
		httpServer: &http.Server{
			Addr:         ":6969",
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

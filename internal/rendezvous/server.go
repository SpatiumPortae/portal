package rendezvous

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/SpatiumPortae/portal/internal/logger"
	"github.com/SpatiumPortae/portal/internal/semver"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Server is contains the necessary data to run the rendezvous server.
type Server struct {
	httpServer *http.Server
	router     *mux.Router
	mailboxes  *Mailboxes
	ids        *IDs
	signal     chan os.Signal
	logger     *zap.Logger
	templates  *template.Template
	version    *semver.Version
}

// NewServer constructs a new Server struct and setups the routes.
func NewServer(port int, version semver.Version) *Server {
	router := &mux.Router{}
	lgr := logger.New()
	stdLoggerWrapper, _ := zap.NewStdLogAt(lgr, zap.ErrorLevel)
	s := &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			Handler:      router,
			ErrorLog:     stdLoggerWrapper,
		},
		router:    router,
		mailboxes: &Mailboxes{&sync.Map{}},
		ids:       &IDs{&sync.Map{}},
		logger:    lgr,
		templates: template.Must(template.ParseGlob("templates/relay/*")),
		version:   &version,
	}
	s.routes()
	return s
}

// Start runs the rendezvous server.
func (s *Server) Start() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-s.signal
		s.logger.Info("portal rendezvous server is shutting down")
		cancel()
	}()

	if err := serve(s, ctx); err != nil {
		s.logger.Error("serving portal rendezvous server", zap.Error(err), zap.Stack("stack_trace"))
	}
}

// serve is a helper function providing graceful shutdown of the server.
func serve(s *Server, ctx context.Context) (err error) {
	go func() {
		if err = s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("serving portal", zap.Error(err), zap.Stack("stack_trace"))
		}
	}()

	s.logger.
		With(zap.String("version", s.version.String())).
		With(zap.String("address", s.httpServer.Addr)).
		Info("serving rendezvous server")
	<-ctx.Done()

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err = s.httpServer.Shutdown(ctxShutdown); err != nil {
		s.logger.Fatal("shutting down rendezvous server", zap.Error(err))
	}

	if err == http.ErrServerClosed {
		err = nil
	}
	s.logger.Info("Portal Rendezvous Server shutdown successfully")
	return err
}

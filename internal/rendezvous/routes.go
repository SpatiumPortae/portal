package rendezvous

import (
	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/logger"
)

func (s *Server) routes() {
	s.router.Use(logger.Middleware(s.logger))
	s.router.HandleFunc("/", s.handleLandingPage())
	s.router.HandleFunc("/ping", s.ping())
	s.router.HandleFunc("/version", s.handleVersionCheck())

	portal := s.router.PathPrefix("").Subrouter()
	portal.Use(conn.Middleware())
	portal.HandleFunc("/establish-sender", s.handleEstablishSender())
	portal.HandleFunc("/establish-receiver", s.handleEstablishReceiver())
}

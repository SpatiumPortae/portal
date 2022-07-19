package rendezvous

import (
	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/logger"
)

func (s *Server) routes() {
	s.router.HandleFunc("/ping", s.ping())
	portal := s.router.PathPrefix("").Subrouter()
	portal.Use(logger.Middleware(s.logger))
	portal.Use(conn.Middleware())
	portal.HandleFunc("/establish-sender", s.handleEstablishSender())
	portal.HandleFunc("/establish-receiver", s.handleEstablishReceiver())
}

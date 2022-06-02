package rendezvous

import "github.com/SpatiumPortae/portal/internal/conn"

func (s *Server) routes() {
	s.router.Use(conn.Middleware())
	s.router.HandleFunc("/establish-sender", s.handleEstablishSender())
	s.router.HandleFunc("/establish-receiver", s.handleEstablishReceiver())
}

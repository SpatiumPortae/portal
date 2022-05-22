package rendezvous

import "github.com/SpatiumPortae/portal/tools"

func (s *Server) routes() {
	s.router.Use(tools.WebsocketMiddleware())
	s.router.HandleFunc("/establish-sender", s.handleEstablishSender())
	s.router.HandleFunc("/establish-receiver", s.handleEstablishReceiver())
}

package server

func (s *Server) routes() {
	s.router.HandleFunc("/establish-sender", s.handleEstablishSender())
}

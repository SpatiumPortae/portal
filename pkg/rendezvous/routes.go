package rendezvous

import "github.com/SpatiumPortae/portal/tools"

func (s *Server) routes() {
	s.router.HandleFunc("/establish-sender", tools.WebsocketHandler(s.handleEstablishSender()))
	s.router.HandleFunc("/establish-receiver", tools.WebsocketHandler(s.handleEstablishReceiver()))
}

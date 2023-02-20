package rendezvous

import (
	"github.com/SpatiumPortae/portal/internal/cache"
	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/logger"
)

func (s *Server) routes() {
	s.router.HandleFunc("/ping", s.ping())

	portal := s.router.PathPrefix("").Subrouter()
	portal.Use(logger.Middleware(s.logger))

	portal.Handle("/version", cache.Middleware(s.storage, "1h", s.handleVersionCheck()))

	portal.Handle("/establish-sender", conn.Middleware()(s.handleEstablishSender()))
	portal.Handle("/establish-receiver", conn.Middleware()(s.handleEstablishReceiver()))
}

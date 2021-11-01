// handlers.go implements the logic for the transfer protocol in the handleTransfer handler.
package sender

import (
	"fmt"
	"net"
	"net/http"
	"syscall"
)

// handleTransfer creates a HandlerFunc to handle serving the transfer of files over a websocket connection
func (s *Sender) handleTransfer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if s.receiverAddr.Equal(net.ParseIP(r.RemoteAddr)) {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "No Portal for You!")
			s.logger.Printf("Unauthorized Portal attempt from alien species with IP: %s\n", r.RemoteAddr)
			return
		}

		wsConn, err := s.senderServer.upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.logger.Printf("Unable to initialize Portal due to technical error: %s\n", err)
			s.done <- syscall.SIGTERM
			return
		}

		s.Transfer(wsConn)
	}
}

package sender

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models/protocol"
)

// handleTransfer creates a HandlerFunc to handle the transfer of files over a websocket.
func (s *Server) handleTransfer() http.HandlerFunc {
	state := Initial
	updateUI(s.ui, state)
	return func(w http.ResponseWriter, r *http.Request) {
		// In case we havce a ui channel, we defer close.
		if s.ui != nil {
			defer close(s.ui)
		}
		// Check if the client has correct address.
		if s.receiverAddr.Equal(net.ParseIP(r.RemoteAddr)) {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "No Portal for You!")
			s.logger.Printf("Unauthorized Portal attempt from alien species with IP: %s.\n", r.RemoteAddr)
			return
		}

		// Establish websocket connection.
		wsConn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.logger.Printf("Unable to initialize Portal due to technical error: %s.\n", err)
			s.done <- syscall.SIGTERM
			return
		}
		defer wsConn.Close()
		s.logger.Printf("Established Potal connection with alien species with IP: %s.\n", r.RemoteAddr)
		state = WaitForHandShake

		for {
			msg := &protocol.TransferMessage{}
			err := wsConn.ReadJSON(msg)

			if err != nil {
				s.logger.Printf("Shutting down portal due to websocket error: %s", err)
				wsConn.Close()
				s.done <- syscall.SIGTERM
				return
			}

			s.logger.Println(*msg)

			//TODO: (ARVID) Implement the exchange of payload size in handshake
			//TODO: (ARVID) Remove unnecessary messaging
			switch msg.Type {
			case protocol.ReceiverHandshake:

				if stateOutOfSync(wsConn, state, WaitForHandShake) {
					s.logger.Println("Sutting down portal due to unsynchronized messaging.")
					wsConn.Close()
					s.done <- syscall.SIGTERM
					return
				}

				wsConn.WriteJSON(protocol.TransferMessage{
					Type:    protocol.SenderHandshake,
					Message: "Portal initialized.",
				})
				state = WaitForFileRequest

			case protocol.ReceiverRequestPayload:
				if stateOutOfSync(wsConn, state, WaitForFileRequest) {
					s.logger.Println("Sutting down portal due to unsynchronized messaging.")
					wsConn.Close()
					s.done <- syscall.SIGTERM
					return
				}
				// TODO: Figure out better size for maximum payload size, static or dynamic?
				buffered := bufio.NewReader(s.payload)
				b := make([]byte, 1024)
				for {
					n, err := buffered.Read(b)
					wsConn.WriteMessage(websocket.BinaryMessage, b[:n]) //TODO: handle error?
					updateUI(s.ui, state, n)
					if err == io.EOF {
						break
					}
				}

				wsConn.WriteJSON(protocol.TransferMessage{
					Type:    protocol.SenderPayloadSent,
					Message: "Portal transfer completed.",
				})
				state = WaitForFileAck
				updateUI(s.ui, state)

			case protocol.ReceiverAckPayload:
				if stateOutOfSync(wsConn, state, WaitForFileAck) {
					s.logger.Println("Sutting down portal due to unsynchronized messaging.")
					wsConn.Close()
					s.done <- syscall.SIGTERM
					return
				}
				state = WaitForCloseMessage

			case protocol.ReceiverClosing:
				if stateOutOfSync(wsConn, state, WaitForCloseMessage) {
					s.logger.Println("Sutting down portal due to unsynchronized messaging.")
					wsConn.Close()
					s.done <- syscall.SIGTERM
					return
				}

				wsConn.WriteJSON(protocol.TransferMessage{
					Type:    protocol.SenderClosing,
					Message: "Closing down the Portal, as requested.",
				})
				state = WaitForCloseAck

			case protocol.ReceiverClosingAck:
				if state != WaitForCloseAck {
					s.logger.Println("Sutting down portal due to unsynchronized messaging.")
				}
				wsConn.Close()
				s.done <- syscall.SIGTERM
				return

			case protocol.TransferError:
				updateUI(s.ui, state)
				s.logger.Printf("Shutting down Portal due to Alien error")
				wsConn.Close()
				s.done <- syscall.SIGTERM
				return
			}
		}
	}
}

// stateOutOfSync is a helper that
func stateOutOfSync(wsConn *websocket.Conn, state, expected TransferState) bool {
	synced := state == expected

	if !synced {
		wsConn.WriteJSON(protocol.TransferMessage{
			Type:    protocol.TransferError,
			Message: "Portal unsynchronized, sutting down.",
		})
	}
	return !synced
}

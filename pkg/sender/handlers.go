package sender

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"syscall"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models/protocol"
)

// handleTransfer creates a HandlerFunc to handle the transfer of files over a websocket.
func (s *Server) handleTransfer() http.HandlerFunc {
	state := Initial
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if the client has correct address.
		if s.receiverAddr.Equal(net.ParseIP(r.RemoteAddr)) {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "No Portal for You!")
			log.Printf("Unauthorized Portal attempt from alien species with IP: %s.\n", r.RemoteAddr)
			return
		}

		// Establish websocket connection.
		wsConn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.logger.Printf("Unable to initialize Portal due to technical error: %s.\n", err)
			s.done <- syscall.SIGTERM
			return
		}
		s.logger.Printf("Established Portal connection with alien species with IP: %s.\n", r.RemoteAddr)
		state = WaitForHandShake

		defer wsConn.Close()

		for {
			msg := &protocol.TransferMessage{}
			err := wsConn.ReadJSON(msg)

			if err != nil {
				s.logger.Printf("Shutting down portal due to websocket error: %s", err)
				s.done <- syscall.SIGTERM
				return
			}

			s.logger.Println(*msg)

			switch msg.Type {
			case protocol.ReceiverHandshake:

				if state != WaitForHandShake {
					wsConn.WriteJSON(protocol.TransferMessage{
						Type:    protocol.TransferError,
						Message: "Portal unsynchronized, shutting down.",
					})
					s.logger.Println("Shutting down portal due to unsynchronized messaging.")
					s.done <- syscall.SIGTERM
					return
				}

				wsConn.WriteJSON(protocol.TransferMessage{
					Type:    protocol.SenderHandshake,
					Message: "Portal initialized.",
				})
				state = WaitForFileRequest

			case protocol.ReceiverRequestPayload:
				if state != WaitForFileRequest {
					wsConn.WriteJSON(protocol.TransferMessage{
						Type:    protocol.TransferError,
						Message: "Portal unsynchronized, shutting down.",
					})
					s.logger.Println("Shutting down portal due to unsynchronized messaging.")
					s.done <- syscall.SIGTERM
				}
				s := bufio.NewScanner(s.payload)
				for s.Scan() {
					wsConn.WriteMessage(websocket.BinaryMessage, s.Bytes()) //TODO: handle error
				}
				wsConn.WriteJSON(protocol.TransferMessage{
					Type:    protocol.SenderPayloadSent,
					Message: "Portal transfer completed.",
				})
				state = WaitForFileAck

			case protocol.ReceiverAckPayload:
				if state != WaitForFileAck {
					wsConn.WriteJSON(protocol.TransferMessage{
						Type:    protocol.TransferError,
						Message: "Portal unsynchronized, shutting down.",
					})
					s.logger.Println("Shutting down portal due to unsynchronized messaging.")
					s.done <- syscall.SIGTERM
					return
				}
				state = WaitForCloseMessage
				// handle multiple payloads.

			case protocol.ReceiverClosing:
				if state != WaitForCloseMessage {
					wsConn.WriteJSON(protocol.TransferMessage{
						Type:    protocol.TransferError,
						Message: "Portal unsynchronized, shutting down.",
					})
					s.logger.Println("Shutting down portal due to unsynchronized messaging.")
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
					s.logger.Println("Shutting down portal due to unsynchronized messaging.")
				}
				s.done <- syscall.SIGTERM
				return

			case protocol.TransferError:
				s.logger.Printf("Shutting down Portal due to Alien error")
				s.done <- syscall.SIGTERM
				return
			}
		}
	}
}

func (s *Server) handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Pong")
	}
}

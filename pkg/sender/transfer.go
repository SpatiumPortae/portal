package sender

import (
	"bufio"
	"io"
	"syscall"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models/protocol"
)

func (s *Sender) Transfer(wsConn *websocket.Conn) error {

	if s.ui != nil {
		defer close(s.ui)
	}

	s.state = WaitForHandShake
	s.updateUI()

	// messaging loop (with state variables)
	for {
		msg := &protocol.TransferMessage{}
		err := wsConn.ReadJSON(msg)
		if err != nil {
			s.logger.Printf("Shutting down portal due to websocket error: %s", err)
			wsConn.Close()
			s.closeServer <- syscall.SIGTERM
			return nil
		}

		switch msg.Type {

		case protocol.ReceiverHandshake:
			if !stateInSync(wsConn, s.state, WaitForHandShake) {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
				wsConn.Close()
				s.closeServer <- syscall.SIGTERM
				return nil
			}

			wsConn.WriteJSON(protocol.TransferMessage{
				Type: protocol.SenderHandshake,
				Payload: protocol.SenderHandshakePayload{
					PayloadSize: s.payloadSize,
				},
			})
			s.state = WaitForFileRequest

		case protocol.ReceiverRequestPayload:
			if !stateInSync(wsConn, s.state, WaitForFileRequest) {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
				wsConn.Close()
				s.closeServer <- syscall.SIGTERM
				return nil
			}
			buffered := bufio.NewReader(s.payload)
			chunkSize := getChunkSize(s.payloadSize)
			b := make([]byte, chunkSize)
			var bytesSent int
			for {
				n, err := buffered.Read(b)
				bytesSent += n
				wsConn.WriteMessage(websocket.BinaryMessage, b[:n]) //TODO: handle error?
				progress := float32(bytesSent) / float32(s.payloadSize)
				s.updateUI(progress)
				if err == io.EOF {
					break
				}
			}
			wsConn.WriteJSON(protocol.TransferMessage{
				Type:    protocol.SenderPayloadSent,
				Payload: "Portal transfer completed",
			})
			s.state = WaitForFileAck
			s.updateUI()

		case protocol.ReceiverPayloadAck:
			if !stateInSync(wsConn, s.state, WaitForFileAck) {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
				wsConn.Close()
				s.closeServer <- syscall.SIGTERM
				return nil
			}
			s.state = WaitForCloseMessage
			wsConn.WriteJSON(protocol.TransferMessage{
				Type:    protocol.SenderClosing,
				Payload: "Closing down the Portal, as requested",
			})
			s.state = WaitForCloseAck

		case protocol.ReceiverClosingAck:
			if s.state != WaitForCloseAck {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
			}
			wsConn.Close()
			s.closeServer <- syscall.SIGTERM
			return nil

		case protocol.TransferError:
			s.updateUI()
			s.logger.Printf("Shutting down Portal due to Alien error")
			wsConn.Close()
			s.closeServer <- syscall.SIGTERM
			return nil
		}
	}
}

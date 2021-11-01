package sender

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"syscall"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func (s *Sender) Transfer(wsConn *websocket.Conn) error {

	if s.ui != nil {
		defer close(s.ui)
	}

	state := WaitForHandShake
	updateUI(s.ui, state)

	// messaging loop (with state variables).
	for {
		msg := &protocol.TransferMessage{}
		err := wsConn.ReadJSON(msg)
		if err != nil {
			s.logger.Printf("Shutting down portal due to websocket error: %s", err)
			wsConn.Close()
			s.done <- syscall.SIGTERM
			return nil
		}

		switch msg.Type {

		case protocol.ReceiverHandshake:
			if !stateInSync(wsConn, state, WaitForHandShake) {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
				wsConn.Close()
				s.done <- syscall.SIGTERM
				return nil
			}

			wsConn.WriteJSON(protocol.TransferMessage{
				Type: protocol.SenderHandshake,
				Payload: protocol.SenderHandshakePayload{
					PayloadSize: s.payloadSize,
				},
			})
			state = WaitForFileRequest

		case protocol.ReceiverRequestPayload:
			if !stateInSync(wsConn, state, WaitForFileRequest) {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
				wsConn.Close()
				s.done <- syscall.SIGTERM
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
				updateUI(s.ui, state, progress)
				if err == io.EOF {
					break
				}
			}
			wsConn.WriteJSON(protocol.TransferMessage{
				Type:    protocol.SenderPayloadSent,
				Payload: "Portal transfer completed",
			})
			state = WaitForFileAck
			updateUI(s.ui, state)

		case protocol.ReceiverAckPayload:
			if !stateInSync(wsConn, state, WaitForFileAck) {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
				wsConn.Close()
				s.done <- syscall.SIGTERM
				return nil
			}
			state = WaitForCloseMessage
			wsConn.WriteJSON(protocol.TransferMessage{
				Type:    protocol.SenderClosing,
				Payload: "Closing down the Portal, as requested",
			})
			state = WaitForCloseAck

		case protocol.ReceiverClosingAck:
			if state != WaitForCloseAck {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
			}
			wsConn.Close()
			s.done <- syscall.SIGTERM
			return nil

		case protocol.TransferError:
			updateUI(s.ui, state)
			s.logger.Printf("Shutting down Portal due to Alien error")
			wsConn.Close()
			s.done <- syscall.SIGTERM
			return nil
		}
	}

}

func ConnectToRendezvous(passwordCh chan<- models.Password, senderReadyCh <-chan bool) (int, net.IP, error) {

	defer close(passwordCh)
	ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s:%s/establish-sender", DEFAULT_RENDEVOUZ_ADDRESS, DEFAULT_RENDEVOUZ_PORT), nil)
	if err != nil {
		return 0, nil, err
	}

	senderPort, err := tools.GetOpenPort()
	if err != nil {
		return 0, nil, err
	}

	ws.WriteJSON(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousEstablish,
		Payload: protocol.SenderToRendezvousEstablishPayload{
			DesiredPort: senderPort,
		},
	})

	msg := protocol.RendezvousMessage{}
	err = ws.ReadJSON(&msg)
	if err != nil {
		return 0, nil, err
	}
	passwordPayload := protocol.RendezvousToSenderGeneratedPasswordPayload{}
	err = tools.DecodePayload(msg.Payload, &passwordPayload)
	if err != nil {
		return 0, nil, err
	}

	// inform about password
	passwordCh <- passwordPayload.Password
	// wait for file preparations to be ready
	<-senderReadyCh

	ws.WriteJSON(protocol.RendezvousMessage{Type: protocol.SenderToRendezvousReady})

	msg = protocol.RendezvousMessage{}
	err = ws.ReadJSON(&msg)
	if err != nil {
		return 0, nil, err
	}
	approvePayload := protocol.RendezvousToSenderApprovePayload{}
	err = tools.DecodePayload(msg.Payload, &approvePayload)

	return senderPort, approvePayload.ReceiverIP, err
}

// stateInSync is a helper that checks the states line up, and reports errors to the receiver in case the states are out of sync
func stateInSync(wsConn *websocket.Conn, state, expected TransferState) bool {
	synced := state == expected
	if !synced {
		wsConn.WriteJSON(protocol.TransferMessage{
			Type:    protocol.TransferError,
			Payload: "Portal unsynchronized, shutting down",
		})
	}
	return synced
}

// updateUI is a helper function that checks if we have a UI channel and reports the state.
func updateUI(ui chan<- UIUpdate, state TransferState, progress ...float32) {
	if ui == nil {
		return
	}
	var p float32
	if len(progress) > 0 {
		p = progress[0]
	}
	ui <- UIUpdate{State: state, Progress: p}
}

// getChunkSize returns an appropriate chunk size for the payload size
func getChunkSize(payloadSize int64) int64 {
	// clamp amount of chunks to be at most MAX_SEND_CHUNKS if it exceeds
	if payloadSize/MAX_CHUNK_BYTES > MAX_SEND_CHUNKS {
		return int64(payloadSize) / MAX_SEND_CHUNKS
	}
	// if not exceeding MAX_SEND_CHUNKS, divide up no. of chunks to MAX_CHUNK_BYTES-sized chunks
	chunkSize := int64(payloadSize) / MAX_CHUNK_BYTES
	// clamp amount of chunks to be at least MAX_CHUNK_BYTES
	if chunkSize <= MAX_CHUNK_BYTES {
		return MAX_CHUNK_BYTES
	}
	return chunkSize
}

package sender

import (
	"bufio"
	"fmt"
	"io"
	"syscall"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func (s *Sender) Transfer(wsConn *websocket.Conn) error {

	s.state = WaitForFileRequest
	for {
		receivedMsg, err := tools.ReadEncryptedMessage(wsConn, s.crypt)
		if err != nil {
			wsConn.Close()
			s.closeServer <- syscall.SIGTERM
			return fmt.Errorf("shutting down portal due to websocket error: %s", err)
		}
		s.logger.Println(receivedMsg.Type.Name())

		switch receivedMsg.Type {
		case protocol.ReceiverRequestPayload:
			if s.state != WaitForFileRequest {
				err = tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{
					Type:    protocol.TransferError,
					Payload: "Portal unsynchronized, shutting down",
				}, s.crypt)
				if err != nil {
					return err
				}
				wsConn.Close()
				s.closeServer <- syscall.SIGTERM
				return NewWrongStateError(WaitForFileRequest, s.state)
			}

			err = s.streamPayload(wsConn)
			if err != nil {
				return err
			}

			err = tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{
				Type:    protocol.SenderPayloadSent,
				Payload: "Portal transfer completed",
			}, s.crypt)
			if err != nil {
				return err
			}

			s.state = WaitForFileAck
			s.updateUI()

		case protocol.ReceiverPayloadAck:
			if s.state != WaitForFileAck {
				err = tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{
					Type:    protocol.TransferError,
					Payload: "Portal unsynchronized, shutting down",
				}, s.crypt)
				if err != nil {
					return err
				}
				wsConn.Close()
				s.closeServer <- syscall.SIGTERM
				return NewWrongStateError(WaitForFileAck, s.state)
			}
			s.state = WaitForCloseMessage
			s.updateUI()

			err = tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{
				Type:    protocol.SenderClosing,
				Payload: "Closing down the Portal, as requested",
			}, s.crypt)
			if err != nil {
				return err
			}
			s.state = WaitForCloseAck
			s.updateUI()

		case protocol.ReceiverClosingAck:
			if s.state != WaitForCloseAck {
				wsConn.Close()
				s.closeServer <- syscall.SIGTERM
				return NewWrongStateError(WaitForCloseAck, s.state)
			}
			wsConn.Close()
			s.closeServer <- syscall.SIGTERM
			return nil

		case protocol.TransferError:
			s.updateUI()
			s.logger.Printf("Shutting down Portal due to Alien error")
			wsConn.Close()
			s.closeServer <- syscall.SIGTERM
			return fmt.Errorf("TransferError during file transfer")
		}
	}
}

func (s *Sender) streamPayload(wsConn *websocket.Conn) error {
	buffered := bufio.NewReader(s.payload)
	chunkSize := getChunkSize(s.payloadSize)
	b := make([]byte, chunkSize)
	var bytesSent int
	for {
		n, err := buffered.Read(b)
		bytesSent += n
		enc, encErr := s.crypt.Encrypt(b[:n])
		if encErr != nil {
			return encErr
		}
		wsConn.WriteMessage(websocket.BinaryMessage, enc)
		progress := float32(bytesSent) / float32(s.payloadSize)
		s.updateUI(progress)
		if err == io.EOF {
			break
		}
	}
	return nil
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

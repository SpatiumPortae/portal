package receiver

import (
	"bytes"
	"encoding/json"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

// TODO: take in expected payload size and progressUpdateCh
func (r *Receiver) Receive(wsConn *websocket.Conn, expectedPayloadSize int64, progressUpdateCh chan<- float32) (*bytes.Buffer, error) {

	if r.ui != nil {
		defer close(r.ui)
	}

	// request payload
	tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{Type: protocol.ReceiverRequestPayload}, r.crypt)

	receivedBuffer := &bytes.Buffer{}
	for {
		msgType, encBytes, err := wsConn.ReadMessage()
		if err != nil {
			return nil, err
		}

		decBytes, err := r.crypt.Decrypt(encBytes)
		if err != nil {
			return nil, err
		}

		if msgType == websocket.BinaryMessage {
			receivedBuffer.Write(decBytes)
			// TODO: what happens when we have no ui channel?
			progressUpdateCh <- float32(receivedBuffer.Len()) / float32(expectedPayloadSize)
		} else {
			transferMsg := protocol.TransferMessage{}
			err = json.Unmarshal(decBytes, &transferMsg)
			if err != nil {
				return nil, err
			}
			if transferMsg.Type != protocol.SenderPayloadSent {
				return nil, protocol.NewWrongMessageTypeError([]protocol.TransferMessageType{protocol.SenderPayloadSent}, transferMsg.Type)
			}
			break
		}
	}

	// ACK received payload
	tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{Type: protocol.ReceiverPayloadAck}, r.crypt)

	transferMsg, err := tools.ReadEncryptedMessage(wsConn, r.crypt)
	if err != nil {
		return nil, err
	}
	if transferMsg.Type != protocol.SenderClosing {
		return nil, protocol.NewWrongMessageTypeError([]protocol.TransferMessageType{protocol.SenderClosing}, transferMsg.Type)
	}

	// ACK SenderClosing with ReceiverClosing
	tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{Type: protocol.ReceiverClosingAck}, r.crypt)

	return receivedBuffer, err
}

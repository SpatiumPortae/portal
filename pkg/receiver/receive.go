package receiver

import (
	"bytes"
	"encoding/json"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func (r *Receiver) Receive(wsConn *websocket.Conn) (*bytes.Buffer, error) {
	if r.ui != nil {
		defer close(r.ui)
	}

	// request payload
	tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{Type: protocol.ReceiverRequestPayload}, r.crypt)

	receivedBuffer := &bytes.Buffer{}
	for {
		_, encBytes, err := wsConn.ReadMessage()
		if err != nil {
			return nil, err
		}

		decBytes, err := r.crypt.Decrypt(encBytes)
		if err != nil {
			return nil, err
		}

		transferMsg := protocol.TransferMessage{}
		err = json.Unmarshal(decBytes, &transferMsg)
		if err != nil {
			receivedBuffer.Write(decBytes)
			r.updateUI(float32(receivedBuffer.Len()) / float32(r.payloadSize))
		} else {
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

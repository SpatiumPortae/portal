package receiver

import (
	"encoding/json"
	"io"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func (r *Receiver) Receive(wsConn *websocket.Conn, buffer io.Writer) error {
	// request payload
	tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{Type: protocol.ReceiverRequestPayload}, r.crypt)

	var writtenBytes int64
	for {
		_, encBytes, err := wsConn.ReadMessage()
		if err != nil {
			return err
		}

		decBytes, err := r.crypt.Decrypt(encBytes)
		if err != nil {
			return err
		}

		transferMsg := protocol.TransferMessage{}
		err = json.Unmarshal(decBytes, &transferMsg)
		if err != nil {
			buffer.Write(decBytes)
			writtenBytes += int64(len(decBytes))
			r.updateUI(float32(writtenBytes) / float32(r.payloadSize))
		} else {
			if transferMsg.Type != protocol.SenderPayloadSent {
				return protocol.NewWrongMessageTypeError([]protocol.TransferMessageType{protocol.SenderPayloadSent}, transferMsg.Type)
			}
			break
		}
	}

	// ACK received payload
	tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{Type: protocol.ReceiverPayloadAck}, r.crypt)

	transferMsg, err := tools.ReadEncryptedMessage(wsConn, r.crypt)
	if err != nil {
		return err
	}
	if transferMsg.Type != protocol.SenderClosing {
		return protocol.NewWrongMessageTypeError([]protocol.TransferMessageType{protocol.SenderClosing}, transferMsg.Type)
	}

	// ACK SenderClosing with ReceiverClosing
	tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{Type: protocol.ReceiverClosingAck}, r.crypt)

	return err
}

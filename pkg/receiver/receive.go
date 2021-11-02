package receiver

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func (r *Receiver) Receive(wsConn *websocket.Conn) error {

	if r.ui != nil {
		defer close(r.ui)
	}

	// request payload
	tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{Type: protocol.ReceiverRequestPayload}, r.crypt)

	for {
		msgType, encBytes, err := wsConn.ReadMessage()
		if err != nil {
			return err
		}

		decBytes, err := r.crypt.Decrypt(encBytes)
		if err != nil {
			return err
		}

		if msgType == websocket.BinaryMessage {
			// write to bytes
		} else {
			transferMsg := protocol.TransferMessage{}
			err = json.Unmarshal(decBytes, &transferMsg)
			if err != nil {
				return err
			}
			if transferMsg.Type != protocol.SenderPayloadSent {
				return fmt.Errorf("Expected")
			}
		}

	}
}

func isExpected(actual protocol.TransferMessageType, expected protocol.TransferMessageType) bool {
	wasExpected := actual == expected
	if !wasExpected {
		log.Printf("Expected message of type: %d. Got type %d\n", expected, actual)
	}
	return wasExpected
}

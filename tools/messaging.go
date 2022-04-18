package tools

import (
	"encoding/json"
	"fmt"

	"github.com/ZinoKader/portal/models/protocol"
	"github.com/ZinoKader/portal/pkg/crypt"
	"github.com/gorilla/websocket"
)

func ReadRendevouzMessage(wsConn *websocket.Conn, expected protocol.RendezvousMessageType) (protocol.RendezvousMessage, error) {
	msg := protocol.RendezvousMessage{}
	err := wsConn.ReadJSON(&msg)
	if err != nil {
		return protocol.RendezvousMessage{}, err
	}
	if msg.Type != expected {
		return protocol.RendezvousMessage{}, fmt.Errorf("expected message type: %d. Got type: %d", expected, msg.Type)
	}
	return msg, nil
}

func WriteEncryptedMessage(wsConn *websocket.Conn, msg protocol.TransferMessage, crypt *crypt.Crypt) error {
	json, err := json.Marshal(msg)
	if err != nil {
		return nil
	}
	enc, err := crypt.Encrypt(json)
	if err != nil {
		return err
	}
	wsConn.WriteMessage(websocket.BinaryMessage, enc)
	return nil
}

func ReadEncryptedMessage(wsConn *websocket.Conn, crypt *crypt.Crypt) (protocol.TransferMessage, error) {
	_, enc, err := wsConn.ReadMessage()
	if err != nil {
		return protocol.TransferMessage{}, err
	}

	dec, err := crypt.Decrypt(enc)
	if err != nil {
		return protocol.TransferMessage{}, err
	}

	msg := protocol.TransferMessage{}
	err = json.Unmarshal(dec, &msg)
	if err != nil {
		return protocol.TransferMessage{}, err
	}
	return msg, nil
}

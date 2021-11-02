package tools

import (
	"fmt"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models/protocol"
)

func ReadRendevouzMessage(wsConn *websocket.Conn, expected protocol.RendezvousMessageType) (protocol.RendezvousMessage, error) {
	msg := protocol.RendezvousMessage{}
	err := wsConn.ReadJSON(&msg)
	if err != nil {
		return protocol.RendezvousMessage{}, err
	}
	if msg.Type != expected {
		return protocol.RendezvousMessage{}, fmt.Errorf("expected message type: %d. Got type:%d", expected, msg.Type)
	}
	return msg, nil
}

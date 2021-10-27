package server

import (
	"log"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func TestIntegration(t *testing.T) {
	expectedPayload := []byte("Portal this shiiiiet")
	s := NewServer()

	senderServer := httptest.NewServer(tools.WebsocketHandler(s.handleEstablishSender()))
	receiverServer := httptest.NewServer(tools.WebsocketHandler(s.handleEstablishReceiver()))

	// TODO: everything below
	ws, _, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/portal", nil)
	if err != nil {
		log.Println(err)
	}
	t.Run("HandShake", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ClientHandshake, Message: ""})
		msg := &protocol.TransferMessage{}
		err := ws.ReadJSON(msg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.ServerHandshake, msg.Type)
	})
	t.Run("Request", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ClientRequestPayload, Message: ""})
		code, b, err := ws.ReadMessage()
		assert.NoError(t, err)
		assert.Equal(t, websocket.BinaryMessage, code)
		assert.Equal(t, expectedPayload, b)
	})
	t.Run("Closing", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ClientClosing, Message: ""})
		msg := &protocol.TransferMessage{}
		err := ws.ReadJSON(msg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.ServerClosing, msg.Type)
		_, _, err = ws.ReadMessage()
		assert.True(t, websocket.IsUnexpectedCloseError(err)) //TODO: fix closing sequence, should client or server close?
	})
}

package sender

import (
	"log"
	"net"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"www.github.com/ZinoKader/portal/models/protocol"
)

func TestIntegration(t *testing.T) {
	expectedPayload := []byte("Portal this shiiiiet")
	s, err := NewServer(8080, expectedPayload, net.ParseIP("127.0.0.1"))
	if err != nil {
		t.Fail()
	}
	server := httptest.NewServer(s.handleTransfer())

	ws, _, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/portal", nil)
	if err != nil {
		log.Println(err)
	}
	t.Run("HandShake", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ReceiverHandshake, Message: ""})
		msg := &protocol.TransferMessage{}
		err := ws.ReadJSON(msg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.SenderHandshake, msg.Type)
	})
	t.Run("Request", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ReceiverRequestPayload, Message: ""})
		code, b, err := ws.ReadMessage()
		assert.NoError(t, err)
		assert.Equal(t, websocket.BinaryMessage, code)
		assert.Equal(t, expectedPayload, b)
	})
	t.Run("Closing", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ReceiverClosing, Message: ""})
		msg := &protocol.TransferMessage{}
		err := ws.ReadJSON(msg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.SenderClosing, msg.Type)
		_, _, err = ws.ReadMessage()
		assert.True(t, websocket.IsUnexpectedCloseError(err)) //TODO: fix closing sequence, should client or server close?
	})
}

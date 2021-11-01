package sender

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

// Test a positive run through the transfer ptotocol
func TestPositiveIntegration(t *testing.T) {
	// Setup
	expectedPayload := []byte("Portal this shiiiiet")
	buf := bytes.NewBuffer(expectedPayload)
	logger := log.New(os.Stderr, "", log.Default().Flags())

	sender := WithServer(NewSender(buf, int64(buf.Len()), net.ParseIP("127.0.0.1"), logger), 8080)
	server := httptest.NewServer(sender.handleTransfer())

	ws, _, _ := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/portal", nil)

	t.Run("HandShake", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ReceiverHandshake, Payload: ""})
		msg := protocol.TransferMessage{}
		err := ws.ReadJSON(&msg)
		payload := protocol.SenderHandshakePayload{}
		tools.DecodePayload(msg.Payload, &payload)
		assert.NoError(t, err)
		assert.Equal(t, protocol.SenderHandshake, msg.Type)
		assert.Equal(t, payload.PayloadSize, len(expectedPayload))
	})
	t.Run("Request", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ReceiverRequestPayload, Payload: ""})
		out := &bytes.Buffer{}

		msg := &protocol.TransferMessage{}
		for {
			code, b, err := ws.ReadMessage()
			assert.NoError(t, err)
			if code != websocket.BinaryMessage {
				err = json.Unmarshal(b, msg)
				assert.NoError(t, err)
				break
			}
			out.Write(b)
		}
		assert.Equal(t, msg.Type, protocol.SenderPayloadSent)
		assert.Equal(t, expectedPayload, out.Bytes())
	})

	t.Run("Close", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ReceiverAckPayload, Payload: ""})
		msg := &protocol.TransferMessage{}
		err := ws.ReadJSON(msg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.SenderClosing, msg.Type)
	})
	t.Run("CloseAck", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ReceiverClosingAck, Payload: ""})
		_, _, err := ws.ReadMessage()
		assert.True(t, websocket.IsUnexpectedCloseError(err))
	})
}

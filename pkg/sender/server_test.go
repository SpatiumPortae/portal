package sender

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/schollz/pake/v3"
	"github.com/stretchr/testify/assert"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/pkg/crypt"
	"www.github.com/ZinoKader/portal/tools"
)

func TestTransfer(t *testing.T) {
	// Setup
	weak := []byte("Normie")
	expectedPayload := []byte("A frog walks into a bank...")
	buf := bytes.NewBuffer(expectedPayload)

	serverOpts := ServerOptions{receiverIP: net.ParseIP("127.0.0.1"), port: 8080}
	programOpts := models.ProgramOptions{RendezvousAddress: "127.0.0.1", RendezvousPort: 3000}
	sender := New(programOpts, WithServer(serverOpts), WithPayload(buf, int64(buf.Len())))

	senderPake, _ := pake.InitCurve(weak, 0, "p256")
	receiverPake, _ := pake.InitCurve(weak, 1, "p256")
	receiverPake.Update(senderPake.Bytes())
	senderPake.Update(receiverPake.Bytes())

	senderKey, _ := senderPake.SessionKey()
	receiverKey, _ := receiverPake.SessionKey()
	sender.crypt, _ = crypt.New(senderKey)
	receiverCrypt, _ := crypt.New(receiverKey, sender.crypt.Salt)

	server := httptest.NewServer(sender.handleTransfer())
	wsConn, _, _ := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/portal", nil)

	t.Run("Request", func(t *testing.T) {
		request := protocol.TransferMessage{Type: protocol.ReceiverRequestPayload}
		tools.WriteEncryptedMessage(wsConn, request, receiverCrypt)

		out := &bytes.Buffer{}
		msg := &protocol.TransferMessage{}

		for {
			_, enc, err := wsConn.ReadMessage()
			assert.NoError(t, err)
			dec, _ := receiverCrypt.Decrypt(enc)
			err = json.Unmarshal(dec, msg)
			if err == nil {
				assert.Equal(t, msg.Type, protocol.SenderPayloadSent)
				break
			}
			out.Write(dec)
		}
		assert.Equal(t, msg.Type, protocol.SenderPayloadSent)
	})

	t.Run("Close", func(t *testing.T) {
		payloadAck := protocol.TransferMessage{Type: protocol.ReceiverPayloadAck}
		tools.WriteEncryptedMessage(wsConn, payloadAck, receiverCrypt)
		msg, err := tools.ReadEncryptedMessage(wsConn, receiverCrypt)
		assert.NoError(t, err)
		assert.Equal(t, protocol.SenderClosing, msg.Type)
	})
}

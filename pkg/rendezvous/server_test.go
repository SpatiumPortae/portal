package rendezvous

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SpatiumPortae/portal/pkg/crypt"
	"github.com/SpatiumPortae/portal/protocol/rendezvous"
	"github.com/SpatiumPortae/portal/tools"
	"github.com/gorilla/websocket"
	"github.com/schollz/pake"
	"github.com/stretchr/testify/assert"
)

func TestIntegration(t *testing.T) {
	s := NewServer(80)
	h := sha256.New()
	testMessage := []byte("A frog walks into a bank...")
	var passStr string
	var password []byte

	mux := http.NewServeMux()
	mux.HandleFunc("/establish-sender", tools.WebsocketHandler(s.handleEstablishSender()))
	mux.HandleFunc("/establish-receiver", tools.WebsocketHandler(s.handleEstablishReceiver()))
	server := httptest.NewServer(mux)

	senderWsConn, _, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/establish-sender", nil)
	assert.NoError(t, err)

	receiverWsConn, _, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/establish-receiver", nil)
	assert.NoError(t, err)

	t.Run("Bind", func(t *testing.T) {

		msg := rendezvous.Msg{}
		err = senderWsConn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.True(t, isExpected(msg.Type, rendezvous.RendezvousToSenderBind))

		passStr = fmt.Sprintf("%d-Normie", msg.Payload.ID)
		h.Write([]byte(passStr))
		password = h.Sum(nil)

		senderWsConn.WriteJSON(rendezvous.Msg{
			Type: rendezvous.SenderToRendezvousEstablish,
			Payload: rendezvous.Payload{
				Password: string(password),
			},
		})
	})

	receiverWsConn.WriteJSON(rendezvous.Msg{
		Type: rendezvous.ReceiverToRendezvousEstablish,
		Payload: rendezvous.Payload{
			Password: string(password),
		},
	})

	t.Run("RendevouzReady", func(t *testing.T) {
		msg := &rendezvous.Msg{}
		err := senderWsConn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.True(t, isExpected(msg.Type, rendezvous.RendezvousToSenderReady))
	})

	senderPake, _ := pake.InitCurve([]byte(passStr), 0, "p256")
	receiverPake, _ := pake.InitCurve([]byte(passStr), 1, "p256")

	senderWsConn.WriteJSON(rendezvous.Msg{
		Type: rendezvous.SenderToRendezvousPAKE,
		Payload: rendezvous.Payload{
			Bytes: senderPake.Bytes(),
		},
	})

	t.Run("ReceiverPAKE", func(t *testing.T) {
		msg := &rendezvous.Msg{}
		err := receiverWsConn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.True(t, isExpected(msg.Type, rendezvous.RendezvousToReceiverPAKE))

		receiverPake.Update(msg.Payload.Bytes)

		receiverWsConn.WriteJSON(&rendezvous.Msg{
			Type: rendezvous.ReceiverToRendezvousPAKE,
			Payload: rendezvous.Payload{
				Bytes: receiverPake.Bytes(),
			},
		})
	})

	t.Run("SenderPAKE", func(t *testing.T) {
		msg := &rendezvous.Msg{}
		err := senderWsConn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.True(t, isExpected(msg.Type, rendezvous.RendezvousToSenderPAKE))

		senderPake.Update(msg.Payload.Bytes)
	})

	senderKey, _ := senderPake.SessionKey()
	receiverKey, _ := receiverPake.SessionKey()
	senderCrypt, _ := crypt.New(senderKey)
	receiverCrypt := &crypt.Crypt{}
	senderWsConn.WriteJSON(&rendezvous.Msg{
		Type: rendezvous.SenderToRendezvousSalt,
		Payload: rendezvous.Payload{
			Salt: senderCrypt.Salt,
		},
	})

	t.Run("ReceiverSalt", func(t *testing.T) {
		msg := &rendezvous.Msg{}
		err := receiverWsConn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.True(t, isExpected(msg.Type, rendezvous.RendezvousToReceiverSalt))

		assert.Equal(t, senderCrypt.Salt, msg.Payload.Salt)
		receiverCrypt, _ = crypt.New(receiverKey, msg.Payload.Salt)
	})

	enc, _ := receiverCrypt.Encrypt(testMessage)
	receiverWsConn.WriteMessage(websocket.BinaryMessage, enc)

	t.Run("SenderEncrypted", func(t *testing.T) {
		_, enc, err := senderWsConn.ReadMessage()
		assert.NoError(t, err)
		dec, err := senderCrypt.Decrypt(enc)
		assert.NoError(t, err)
		assert.Equal(t, testMessage, dec)
	})

	// TODO: test sender-receiver-matching timeout
}

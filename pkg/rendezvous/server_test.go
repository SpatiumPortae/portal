package rendezvous

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/password"
	"github.com/SpatiumPortae/portal/pkg/crypt"
	"github.com/SpatiumPortae/portal/protocol/rendezvous"
	"github.com/SpatiumPortae/portal/tools"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/schollz/pake"
	"github.com/stretchr/testify/assert"
)

func TestIntegration(t *testing.T) {
	s := NewServer(80)
	testMessage := []byte("A frog walks into a bank...")
	var passStr string
	var hashedPassword string

	mux := mux.NewRouter()
	mux.Use(tools.WebsocketMiddleware())
	mux.HandleFunc("/establish-sender", s.handleEstablishSender())
	mux.HandleFunc("/establish-receiver", s.handleEstablishReceiver())
	server := httptest.NewServer(mux)

	senderWsConn, _, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/establish-sender", nil)
	assert.NoError(t, err)
	senderConn := conn.Rendezvous{Conn: &conn.WS{Conn: senderWsConn}}

	receiverWsConn, _, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/establish-receiver", nil)
	assert.NoError(t, err)
	receiverConn := conn.Rendezvous{Conn: &conn.WS{Conn: receiverWsConn}}

	msg, err := senderConn.ReadMsg(rendezvous.RendezvousToSenderBind)

	assert.NoError(t, err)

	passStr = fmt.Sprintf("%d-Normie", msg.Payload.ID)
	hashedPassword = password.Hashed(passStr)

	senderConn.WriteMsg(rendezvous.Msg{
		Type: rendezvous.SenderToRendezvousEstablish,
		Payload: rendezvous.Payload{
			Password: hashedPassword,
		},
	})

	receiverConn.WriteMsg(rendezvous.Msg{
		Type: rendezvous.ReceiverToRendezvousEstablish,
		Payload: rendezvous.Payload{
			Password: hashedPassword,
		},
	})

	_, err = senderConn.ReadMsg(rendezvous.RendezvousToSenderReady)
	assert.NoError(t, err)

	senderPake, _ := pake.InitCurve([]byte(passStr), 0, "p256")
	receiverPake, _ := pake.InitCurve([]byte(passStr), 1, "p256")

	senderConn.WriteMsg(rendezvous.Msg{
		Type: rendezvous.SenderToRendezvousPAKE,
		Payload: rendezvous.Payload{
			Bytes: senderPake.Bytes(),
		},
	})

	msg, err = receiverConn.ReadMsg(rendezvous.RendezvousToReceiverPAKE)
	assert.NoError(t, err)

	receiverPake.Update(msg.Payload.Bytes)

	receiverConn.WriteMsg(rendezvous.Msg{
		Type: rendezvous.ReceiverToRendezvousPAKE,
		Payload: rendezvous.Payload{
			Bytes: receiverPake.Bytes(),
		},
	})

	msg, err = senderConn.ReadMsg(rendezvous.RendezvousToSenderPAKE)
	assert.NoError(t, err)

	senderPake.Update(msg.Payload.Bytes)

	senderKey, _ := senderPake.SessionKey()
	receiverKey, _ := receiverPake.SessionKey()
	senderCrypt, _ := crypt.New(senderKey)

	senderConn.WriteMsg(rendezvous.Msg{
		Type: rendezvous.SenderToRendezvousSalt,
		Payload: rendezvous.Payload{
			Salt: senderCrypt.Salt,
		},
	})

	msg, err = receiverConn.ReadMsg(rendezvous.RendezvousToReceiverSalt)
	assert.NoError(t, err)

	assert.Equal(t, senderCrypt.Salt, msg.Payload.Salt)
	receiverCrypt, _ := crypt.New(receiverKey, msg.Payload.Salt)

	enc, _ := receiverCrypt.Encrypt(testMessage)
	receiverConn.Conn.Write(enc) // send raw bytes

	enc, err = senderConn.Conn.Read() // read raw bytes
	assert.NoError(t, err)
	dec, err := senderCrypt.Decrypt(enc)
	assert.NoError(t, err)
	assert.Equal(t, testMessage, dec)

	// TODO: test sender-receiver-matching timeout
}

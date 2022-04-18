package sender

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"

	"github.com/gorilla/websocket"
	"github.com/schollz/pake/v3"
	"www.github.com/ZinoKader/portal/internal/conn"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func ConnectRendezvous(addr net.TCPAddr) (conn.RendezvousConn, string, error) {
	ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/establish-sender", addr.String()), nil)
	if err != nil {
		return conn.RendezvousConn{}, "", err
	}

	rc := conn.RendezvousConn{Conn: &conn.WS{Conn: ws}}

	msg, err := rc.ReadMsg(protocol.RendezvousToSenderBind)
	if err != nil {
		return conn.RendezvousConn{}, "", err
	}
	bind := msg.Payload.(protocol.RendezvousToSenderBindPayload)
	password := tools.GeneratePassword(bind.ID)
	hashed := tools.HashPassword(password)

	err = rc.WriteMsg(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousEstablish,
		Payload: protocol.PasswordPayload{
			Password: hashed,
		},
	})
	if err != nil {
		return conn.RendezvousConn{}, "", err
	}
	return rc, string(password), nil
}

func SecureConnection(rc conn.RendezvousConn, password string) (conn.TransferConn, error) {
	pake, err := pake.InitCurve([]byte(password), 0, "p256")
	if err != nil {
		return conn.TransferConn{}, err
	}
	// Wait for for the receiver to be ready.
	_, err = rc.ReadMsg(protocol.RendezvousToSenderReady)
	if err != nil {
		return conn.TransferConn{}, err
	}
	// Start the key exchange.
	err = rc.WriteMsg(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousPAKE,
		Payload: protocol.PakePayload{
			Bytes: pake.Bytes(),
		},
	})

	if err != nil {
		return conn.TransferConn{}, err
	}
	msg, err := rc.ReadMsg()
	if err != nil {
		return conn.TransferConn{}, err
	}
	payload := msg.Payload.(protocol.PakePayload)
	if err := pake.Update(payload.Bytes); err != nil {
		return conn.TransferConn{}, err
	}

	// create salt and session key.
	salt := make([]byte, 8)
	if _, err := rand.Read(salt); err != nil {
		return conn.TransferConn{}, err
	}
	session, err := pake.SessionKey()
	if err != nil {
		return conn.TransferConn{}, err
	}
	err = rc.WriteMsg(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousSalt,
		Payload: protocol.SaltPayload{
			Salt: salt,
		},
	})
	if err != nil {
		return conn.TransferConn{}, err
	}
	return conn.NewTransferConn(rc.Conn, session, salt), nil
}

func Transfer(ctx context.Context, tc conn.TransferConn, payload []byte) error {
	msg, err := tc.ReadMsg(protocol.ReceiverHandshake)
	if err != nil {
		return nil
	}
	recvHandshake := msg.Payload.(protocol.ReceiverHandshakePayload)

	port, err := tools.GetOpenPort()
	if err != nil {
		return err
	}
	server := NewServer(port)
}

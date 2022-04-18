package sender

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
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

func SecureConnection(rc conn.RendezvousConn, password string) (conn.Transfer, error) {
	pake, err := pake.InitCurve([]byte(password), 0, "p256")
	if err != nil {
		return conn.Transfer{}, err
	}
	// Wait for for the receiver to be ready.
	_, err = rc.ReadMsg(protocol.RendezvousToSenderReady)
	if err != nil {
		return conn.Transfer{}, err
	}
	// Start the key exchange.
	err = rc.WriteMsg(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousPAKE,
		Payload: protocol.PakePayload{
			Bytes: pake.Bytes(),
		},
	})

	if err != nil {
		return conn.Transfer{}, err
	}
	msg, err := rc.ReadMsg()
	if err != nil {
		return conn.Transfer{}, err
	}
	payload := msg.Payload.(protocol.PakePayload)
	if err := pake.Update(payload.Bytes); err != nil {
		return conn.Transfer{}, err
	}

	// create salt and session key.
	salt := make([]byte, 8)
	if _, err := rand.Read(salt); err != nil {
		return conn.Transfer{}, err
	}
	session, err := pake.SessionKey()
	if err != nil {
		return conn.Transfer{}, err
	}
	err = rc.WriteMsg(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousSalt,
		Payload: protocol.SaltPayload{
			Salt: salt,
		},
	})
	if err != nil {
		return conn.Transfer{}, err
	}
	return conn.TransferFromSession(rc.Conn, session, salt), nil
}

func Transfer(tc conn.Transfer, payload io.Reader, payloadSize int64, writers ...io.Writer) error {
	_, err := tc.ReadMsg(protocol.ReceiverHandshake)
	if err != nil {
		return nil
	}

	port, err := tools.GetOpenPort()
	if err != nil {
		return err
	}
	server := NewServer(port)

	// Start server for transfers on the same network.
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("%v", err)
		}
	}()
	defer server.Shutdown()

	ip, err := tools.GetLocalIP()
	if err != nil {
		return err
	}
	handshake := protocol.TransferMessage{
		Type: protocol.SenderHandshake,
		Payload: protocol.SenderHandshakePayload{
			IP:          ip,
			Port:        port,
			PayloadSize: payloadSize,
		},
	}
	if err := tc.WriteMsg(handshake); err != nil {
		return err
	}

	msg, err := tc.ReadMsg()
	if err != nil {
		return err
	}

	switch msg.Type {
	case protocol.ReceiverDirectCommunication:
		if err := tc.WriteMsg(protocol.TransferMessage{Type: protocol.SenderDirectAck}); err != nil {
			return err
		}
		// Wait for transfer to complete somehow
		return nil
	case protocol.ReceiverRelayCommunication:
		if err := tc.WriteMsg(protocol.TransferMessage{Type: protocol.SenderRelayAck}); err != nil {
			return err
		}
		return transfer(tc, payload, writers...)
	default:
		return protocol.NewWrongTransferMessageTypeError(
			[]protocol.TransferMessageType{protocol.ReceiverDirectCommunication, protocol.ReceiverRelayCommunication},
			msg.Type)
	}
}

func transfer(tc conn.Transfer, payload io.Reader, writers ...io.Writer) error {
	_, err := tc.ReadMsg(protocol.ReceiverRequestPayload)
	if err != nil {
		return err
	}
	// add our connection to the list of writers, and copy the payload to all writers.
	writers = append(writers, tc)
	_, err = io.Copy(io.MultiWriter(writers...), payload)
	if err != nil {
		return err
	}
	if err := tc.WriteMsg(protocol.TransferMessage{Type: protocol.SenderPayloadSent}); err != nil {
		return err
	}

	_, err = tc.ReadMsg(protocol.ReceiverPayloadAck)
	if err != nil {
		return err
	}

	if err := tc.WriteMsg(protocol.TransferMessage{Type: protocol.SenderClosing}); err != nil {
		return err
	}
	return nil
}

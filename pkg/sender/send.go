package sender

import (
	"bufio"
	"context"
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

type TransferType int

const (
	Direct TransferType = iota
	Relay
)

// ConnectRendezvous creates a connection with the rendezvous server and acquires a password associated with the connection
func ConnectRendezvous(addr net.TCPAddr) (conn.Rendezvous, string, error) {
	ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/establish-sender", addr.String()), nil)
	if err != nil {
		return conn.Rendezvous{}, "", err
	}

	rc := conn.Rendezvous{Conn: &conn.WS{Conn: ws}}

	msg, err := rc.ReadMsg(protocol.RendezvousToSenderBind)
	if err != nil {
		return conn.Rendezvous{}, "", err
	}
	bind := msg.Payload.(protocol.RendezvousToSenderBindPayload)
	password := tools.GeneratePassword(bind.ID)
	hashed := tools.HashPassword(password)

	if err := rc.WriteMsg(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousEstablish,
		Payload: protocol.PasswordPayload{
			Password: hashed,
		},
	}); err != nil {
		return conn.Rendezvous{}, "", err
	}
	return rc, string(password), nil
}

// SecureConnection does the cryptographic handshake in order to resolve a secure channel to do file transfer over.
func SecureConnection(rc conn.Rendezvous, password string) (conn.Transfer, error) {
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

// Transfer preforms the file transfer, either directly or using the Rendezvous server as a relay.
func Transfer(tc conn.Transfer, payload io.Reader, payloadSize int64, msgs ...chan interface{}) error {
	_, err := tc.ReadMsg(protocol.ReceiverHandshake)
	if err != nil {
		return err
	}
	port, err := tools.GetOpenPort()
	if err != nil {
		return err
	}
	server := NewServer(port, tc.Key(), payload, msgs...)

	ctx := context.Background()
	// Start server for direct transfers.
	go func() {
		if err := server.Start(ctx); err != nil {
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

		if len(msgs) > 0 {
			msgs[0] <- Direct
		}
		// Wait for direct transfer to finish.
		<-ctx.Done()
		return server.Err
	case protocol.ReceiverRelayCommunication:
		if err := tc.WriteMsg(protocol.TransferMessage{Type: protocol.SenderRelayAck}); err != nil {
			return err
		}

		if len(msgs) > 0 {
			msgs[0] <- Relay
		}
		return transfer(tc, payload, msgs...)
	default:
		return protocol.NewWrongTransferMessageTypeError(
			[]protocol.TransferMessageType{protocol.ReceiverDirectCommunication, protocol.ReceiverRelayCommunication},
			msg.Type)
	}
}

// transfer is a helper method that actually preforms the transfer sequence.
func transfer(tc conn.Transfer, payload io.Reader, msgs ...chan interface{}) error {
	_, err := tc.ReadMsg(protocol.ReceiverRequestPayload)
	if err != nil {
		return err
	}
	err = transferPayload(tc, payload, msgs...)
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

func transferPayload(tc conn.Transfer, payload io.Reader, msgs ...chan interface{}) error {
	bufReader := bufio.NewReader(payload)
	buffer := make([]byte, 512) // max size of a websocket message
	bytesSent := 0
	for {
		n, err := bufReader.Read(buffer)
		bytesSent += n
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		err = tc.WriteBytes(buffer)
		if err != nil {
			return err
		}

		if len(msgs) > 0 {
			msgs[0] <- bytesSent
		}

	}
	return nil
}

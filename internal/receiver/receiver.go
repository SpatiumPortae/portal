package receiver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/gorilla/websocket"
	"github.com/schollz/pake"
	"www.github.com/ZinoKader/portal/internal/conn"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func ConnectRendezvous(addr net.TCPAddr) (conn.Rendezvous, error) {
	ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/establish-receiver", addr.String()), nil)
	if err != nil {
		return conn.Rendezvous{}, err
	}
	return conn.Rendezvous{Conn: &conn.WS{Conn: ws}}, nil
}

func SecureConnection(rc conn.Rendezvous, password string) (conn.Transfer, error) {

	// Convenience for messaging in this function.
	type pakeMsg struct {
		pake *pake.Pake
		err  error
	}
	pakeCh := make(chan pakeMsg)

	// Init pake curve in background.
	go func() {
		p, err := pake.InitCurve([]byte(password), 1, "p256")
		pakeCh <- pakeMsg{pake: p, err: err}
	}()

	if err := rc.WriteMsg(protocol.RendezvousMessage{
		Type: protocol.ReceiverToRendezvousEstablish,
		Payload: protocol.PasswordPayload{
			Password: tools.HashPassword(models.Password(password)),
		},
	}); err != nil {
		return conn.Transfer{}, err
	}

	msg, err := rc.ReadMsg(protocol.RendezvousToReceiverPAKE)
	if err != nil {
		return conn.Transfer{}, err
	}
	b := msg.Payload.(protocol.PakePayload).Bytes
	pm := <-pakeCh

	if pm.err != nil {
		return conn.Transfer{}, err
	}
	p := pm.pake

	err = p.Update(b)
	if err = rc.WriteMsg(protocol.RendezvousMessage{
		Type: protocol.ReceiverToRendezvousPAKE,
		Payload: protocol.PakePayload{
			Bytes: p.Bytes(),
		},
	}); err != nil {
		return conn.Transfer{}, err
	}

	session, err := p.SessionKey()
	if err != nil {
		return conn.Transfer{}, err
	}

	msg, err = rc.ReadMsg(protocol.RendezvousToReceiverSalt)
	if err != nil {
		return conn.Transfer{}, err
	}
	salt := msg.Payload.(protocol.SaltPayload).Salt

	return conn.TransferFromSession(rc.Conn, session, salt), nil
}

func Receive(tc conn.Transfer, msgs ...chan interface{}) (bytes.Buffer, error) {
	if err := tc.WriteMsg(protocol.TransferMessage{Type: protocol.ReceiverHandshake}); err != nil {
		return bytes.Buffer{}, err
	}

	msg, err := tc.ReadMsg(protocol.SenderHandshake)
	if err != nil {
		return bytes.Buffer{}, err
	}

	payload := msg.Payload.(protocol.SenderHandshakePayload)

	if len(msgs) > 0 {
		msgs[0] <- payload.PayloadSize
	}
	return receive(tc, net.TCPAddr{IP: payload.IP, Port: payload.Port}, msgs...)
}

func receive(relay conn.Transfer, addr net.TCPAddr, msgs ...chan interface{}) (bytes.Buffer, error) {
	var tc conn.Transfer
	direct, err := probeSender(addr, relay.Key())
	if err != nil {
		tc = relay
		if len(msgs) > 0 {
			msgs[0] <- protocol.Relay
		}
	} else {
		tc = direct
		if len(msgs) > 0 {
			msgs[0] <- protocol.Direct
		}
	}

	if tc.WriteMsg(protocol.TransferMessage{Type: protocol.ReceiverRequestPayload}) != nil {
		return bytes.Buffer{}, err
	}
	buffer, err := receivePayload(tc, msgs...)
	if err != nil {
		return bytes.Buffer{}, err
	}

	if err := tc.WriteMsg(protocol.TransferMessage{Type: protocol.ReceiverPayloadAck}); err != nil {
		return bytes.Buffer{}, err
	}

	_, err = tc.ReadMsg(protocol.SenderClosing)

	if err != nil {
		return bytes.Buffer{}, err
	}

	if err := tc.WriteMsg(protocol.TransferMessage{Type: protocol.ReceiverClosingAck}); err != nil {
		return bytes.Buffer{}, err
	}

	return buffer, nil
}

func receivePayload(tc conn.Transfer, msgs ...chan interface{}) (bytes.Buffer, error) {
	buffer := bytes.Buffer{}
	writtenBytes := 0
	for {
		b, err := tc.ReadBytes()
		if err != nil {
			return bytes.Buffer{}, err
		}
		msg := protocol.TransferMessage{}
		err = json.Unmarshal(b, &msg)
		if err != nil {
			n, err := buffer.Write(b)
			if err != nil {
				return bytes.Buffer{}, err
			}
			writtenBytes += n
			if len(msgs) > 0 {
				msgs[0] <- writtenBytes
			}
		} else {
			if msg.Type != protocol.SenderPayloadSent {
				return bytes.Buffer{}, protocol.NewWrongTransferMessageTypeError([]protocol.TransferMessageType{protocol.SenderPayloadSent}, msg.Type)
			}
			break
		}
	}
	return buffer, nil
}

func probeSender(addr net.TCPAddr, key []byte) (conn.Transfer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second) // wait at most 3 seconds.
	defer cancel()
	d := 250 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return conn.Transfer{}, fmt.Errorf("could not establish a connection to the sender server")

		default:
			dialer := websocket.Dialer{HandshakeTimeout: d}
			ws, _, err := dialer.Dial(fmt.Sprintf("ws://%s/portal", addr.String()), nil)
			if err != nil {
				time.Sleep(d)
				d = d * 2
				continue
			}
			return conn.TransferFromKey(&conn.WS{Conn: ws}, key), nil
		}
	}
}

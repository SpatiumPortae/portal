package receiver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"
	"github.com/schollz/pake"
	"www.github.com/ZinoKader/portal/internal/conn"
	"www.github.com/ZinoKader/portal/internal/password"
	"www.github.com/ZinoKader/portal/protocol/rendezvous"
	"www.github.com/ZinoKader/portal/protocol/transfer"
)

// ConnectRendezvous makes the initial connection to the rendezvous server.
func ConnectRendezvous(addr net.TCPAddr) (conn.Rendezvous, error) {
	ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/establish-receiver", addr.String()), nil)
	if err != nil {
		return conn.Rendezvous{}, err
	}
	return conn.Rendezvous{Conn: &conn.WS{Conn: ws}}, nil
}

// SecureConnection performs the cryptographic handshake to resolve a secure connection.
func SecureConnection(rc conn.Rendezvous, pass string) (conn.Transfer, error) {
	// Convenience for messaging in this function.
	type pakeMsg struct {
		pake *pake.Pake
		err  error
	}
	pakeCh := make(chan pakeMsg)

	// Init pake curve in background.
	go func() {
		p, err := pake.InitCurve([]byte(pass), 1, "p256")
		pakeCh <- pakeMsg{pake: p, err: err}
	}()

	if err := rc.WriteMsg(rendezvous.Msg{
		Type: rendezvous.ReceiverToRendezvousEstablish,
		Payload: rendezvous.Payload{
			Password: password.Hashed(pass),
		},
	}); err != nil {
		return conn.Transfer{}, err
	}

	msg, err := rc.ReadMsg(rendezvous.RendezvousToReceiverPAKE)
	if err != nil {
		return conn.Transfer{}, err
	}
	pm := <-pakeCh

	if pm.err != nil {
		return conn.Transfer{}, err
	}
	p := pm.pake

	err = p.Update(msg.Payload.Bytes)
	if err = rc.WriteMsg(rendezvous.Msg{
		Type: rendezvous.ReceiverToRendezvousPAKE,
		Payload: rendezvous.Payload{
			Bytes: p.Bytes(),
		},
	}); err != nil {
		return conn.Transfer{}, err
	}

	session, err := p.SessionKey()
	if err != nil {
		return conn.Transfer{}, err
	}

	msg, err = rc.ReadMsg(rendezvous.RendezvousToReceiverSalt)
	if err != nil {
		return conn.Transfer{}, err
	}

	return conn.TransferFromSession(rc.Conn, session, msg.Payload.Salt), nil
}

// Receive receives the payload over the transfer connection and writes it into the provided destination.
// The Transfer can either be direct or using a relay.
// The msgs channel communicates information about the receiving process while running.
func Receive(tc conn.Transfer, dst io.Writer, msgs ...chan interface{}) error {
	if err := tc.WriteMsg(transfer.Msg{Type: transfer.ReceiverHandshake}); err != nil {
		return err
	}

	msg, err := tc.ReadMsg(transfer.SenderHandshake)
	if err != nil {
		return err
	}

	if len(msgs) > 0 {
		msgs[0] <- msg.Payload.PayloadSize
	}
	return receive(tc, net.TCPAddr{IP: msg.Payload.IP, Port: msg.Payload.Port}, dst, msgs...)
}

// receive preforms the transfer protocol on the receiving end.
func receive(relay conn.Transfer, addr net.TCPAddr, dst io.Writer, msgs ...chan interface{}) error {

	// Retrieve a unencrypted channel to rendezvous.
	rc := conn.Rendezvous{Conn: relay.Conn}
	// Determine if we should do direct or relay transfer.
	var tc conn.Transfer
	direct, err := probeSender(addr, relay.Key())
	if err != nil {
		tc = relay

		// Communicate to the sender that we are using relay transfer.
		if err := relay.WriteMsg(transfer.Msg{Type: transfer.ReceiverRelayCommunication}); err != nil {
			return err
		}
		_, err := relay.ReadMsg(transfer.SenderRelayAck)
		if err != nil {
			return err
		}

		if len(msgs) > 0 {
			msgs[0] <- transfer.Relay
		}
	} else {
		tc = direct
		// Communicate to the sender that we are doing direct communication.
		if err := relay.WriteMsg(transfer.Msg{Type: transfer.ReceiverDirectCommunication}); err != nil {
			return err
		}

		// Tell rendezvous server that we can close the connection.
		if err := rc.WriteMsg(rendezvous.Msg{Type: rendezvous.ReceiverToRendezvousClose}); err != nil {
			return err
		}

		if len(msgs) > 0 {
			msgs[0] <- transfer.Direct
		}
	}

	// Request the payload and receive it.
	if tc.WriteMsg(transfer.Msg{Type: transfer.ReceiverRequestPayload}) != nil {
		return err
	}
	if err := receivePayload(tc, dst, msgs...); err != nil {
		return err
	}

	// Closing handshake.

	if err := tc.WriteMsg(transfer.Msg{Type: transfer.ReceiverPayloadAck}); err != nil {
		return err
	}

	_, err = tc.ReadMsg(transfer.SenderClosing)

	if err != nil {
		return err
	}

	if err := tc.WriteMsg(transfer.Msg{Type: transfer.ReceiverClosingAck}); err != nil {
		return err
	}

	// Tell rendezvous to close connection.
	if err := rc.WriteMsg(rendezvous.Msg{Type: rendezvous.ReceiverToRendezvousClose}); err != nil {
		return err
	}
	return nil
}

// receivePayload receives the payload over the provided connection and writes it into the desired location.
func receivePayload(tc conn.Transfer, dst io.Writer, msgs ...chan interface{}) error {
	writtenBytes := 0
	for {
		b, err := tc.ReadBytes()
		if err != nil {
			return err
		}
		msg := transfer.Msg{}
		err = json.Unmarshal(b, &msg)
		if err != nil {
			n, err := dst.Write(b)
			if err != nil {
				return err
			}
			writtenBytes += n
			if len(msgs) > 0 {
				msgs[0] <- writtenBytes
			}
		} else {
			if msg.Type != transfer.SenderPayloadSent {
				return transfer.Error{Expected: []transfer.MsgType{transfer.SenderPayloadSent}, Got: msg.Type}
			}
			break
		}
	}
	return nil
}

// probeSender will try to connect directly to the sender using a linear back off for up to 3 seconds.
// Returns a transfer connection channel if it succeeds, otherwise it returns an error.
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

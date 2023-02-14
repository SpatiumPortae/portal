package receiver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/password"
	"github.com/SpatiumPortae/portal/protocol/rendezvous"
	"github.com/SpatiumPortae/portal/protocol/transfer"
	"github.com/schollz/pake/v3"
	"nhooyr.io/websocket"
)

// ------------------------------------------------- Receiver Functions ------------------------------------------------

// ConnectRendezvous makes the initial connection to the rendezvous server.
func ConnectRendezvous(addr string) (conn.Rendezvous, error) {
	ws, _, err := websocket.Dial(context.Background(), fmt.Sprintf("ws://%s/establish-receiver", addr), nil)
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
	if err != nil {
		return conn.Transfer{}, err
	}

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

func TransferHandshake(tc conn.Transfer, writers ...io.Writer) (Receiver, error) {
	receiver, err := handshake(tc, writers...)
	if err != nil {
		return nil, err
	}
	return receiver, nil
}

// ------------------------------------------------------ Receiver -----------------------------------------------------

// Receiver represents a entity that can perform the receive.
type Receiver interface {
	Type() transfer.Type
	Receive(dst io.Writer) error
	PayloadSize() int64
}

type receiver struct {
	transferType transfer.Type
	payloadSize  int64
	rc           conn.Rendezvous
	tc           conn.Transfer
	writers      []io.Writer
}

func (r receiver) Receive(dst io.Writer) error {
	if err := r.tc.WriteMsg(transfer.Msg{Type: transfer.ReceiverRequestPayload}); err != nil {
		return err
	}
	writers := append(r.writers, dst)
	if err := receivePayload(r.tc, io.MultiWriter(writers...)); err != nil {
		return fmt.Errorf("receiving encrypted payload: %w", err)
	}
	// Closing handshake.
	if err := r.tc.WriteMsg(transfer.Msg{Type: transfer.ReceiverPayloadAck}); err != nil {
		return err
	}
	if _, err := r.tc.ReadMsg(transfer.SenderClosing); err != nil {
		return err
	}
	if err := r.tc.WriteMsg(transfer.Msg{Type: transfer.ReceiverClosingAck}); err != nil {
		return err
	}
	// Tell rendezvous to close connection.
	if err := r.rc.WriteMsg(rendezvous.Msg{Type: rendezvous.ReceiverToRendezvousClose}); err != nil {
		return err
	}
	return nil
}

func (r receiver) Type() transfer.Type {
	return r.transferType
}

func (r receiver) PayloadSize() int64 {
	return r.payloadSize
}

// ------------------------------------------------------ Helpers ------------------------------------------------------

// receivePayload receives the payload over the provided connection and writes it into the desired location.
func receivePayload(tc conn.Transfer, dst io.Writer) error {
	for {
		b, err := tc.Read()
		if err != nil {
			return err
		}
		msg := transfer.Msg{}
		err = json.Unmarshal(b, &msg)
		if err != nil {
			_, err := dst.Write(b)
			if err != nil {
				return err
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

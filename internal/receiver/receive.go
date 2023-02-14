//go:build !js

package receiver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/protocol/rendezvous"
	"github.com/SpatiumPortae/portal/protocol/transfer"
	"nhooyr.io/websocket"
)

// handshake performs the transfer protocol on the receiving end.
// This function is built for all platforms except js
func handshake(relay conn.Transfer, writers ...io.Writer) (Receiver, error) {
	if err := relay.WriteMsg(transfer.Msg{Type: transfer.ReceiverHandshake}); err != nil {
		return nil, err
	}
	msg, err := relay.ReadMsg(transfer.SenderHandshake)
	if err != nil {
		return nil, err
	}

	// Retrieve a unencrypted channel to rendezvous.
	rc := conn.Rendezvous{Conn: relay.Conn}
	// Determine if we should do direct or relay transfer.
	var tc conn.Transfer
	var transferType transfer.Type
	direct, err := probeSender(fmt.Sprintf("%s:%d", msg.Payload.IP, msg.Payload.Port), relay.Key())
	if err != nil {
		tc = relay
		transferType = transfer.Relay
		// Communicate to the sender that we are using relay transfer.
		if err := relay.WriteMsg(transfer.Msg{Type: transfer.ReceiverRelayCommunication}); err != nil {
			return nil, err
		}
		_, err := relay.ReadMsg(transfer.SenderRelayAck)
		if err != nil {
			return nil, err
		}

	} else {
		tc = direct
		transferType = transfer.Direct
		// Communicate to the sender that we are doing direct communication.
		if err := relay.WriteMsg(transfer.Msg{Type: transfer.ReceiverDirectCommunication}); err != nil {
			return nil, err
		}

		// Tell rendezvous server that we can close the connection.
		if err := rc.WriteMsg(rendezvous.Msg{Type: rendezvous.ReceiverToRendezvousClose}); err != nil {
			return nil, err
		}
	}
	return receiver{
		transferType: transferType,
		payloadSize:  msg.Payload.PayloadSize,
		tc:           tc,
		rc:           rc,
		writers:      writers,
	}, nil
}

// probeSender will try to connect directly to the sender using a linear back off for up to 3 seconds.
// Returns a transfer connection channel if it succeeds, otherwise it returns an error.
func probeSender(addr string, key []byte) (conn.Transfer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second) // wait at most 3 seconds.
	defer cancel()
	d := 250 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return conn.Transfer{}, fmt.Errorf("could not establish a connection to the sender server")

		default:
			ws, _, err := websocket.Dial(
				context.Background(), fmt.Sprintf("ws://%s/portal", addr),
				&websocket.DialOptions{HTTPClient: &http.Client{Timeout: d}},
			)
			if err != nil {
				time.Sleep(d)
				d = d * 2
				continue
			}
			return conn.TransferFromKey(&conn.WS{Conn: ws}, key), nil
		}
	}
}

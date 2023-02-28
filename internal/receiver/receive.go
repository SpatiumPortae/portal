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

// doReceive performs the transfer protocol on the receiving end.
// This function is built for all platforms except js
func doReceive(ctx context.Context, relay conn.Transfer, addr string, dst io.Writer, msgs ...chan interface{}) error {

	// Retrieve a unencrypted channel to rendezvous.
	rc := conn.Rendezvous{Conn: relay.Conn}
	// Determine if we should do direct or relay transfer.
	var tc conn.Transfer
	direct, err := probeSender(addr, relay.Key())
	if err != nil {
		tc = relay
		// Communicate to the sender that we are using relay transfer.
		if err := relay.WriteMsg(ctx, transfer.Msg{Type: transfer.ReceiverRelayCommunication}); err != nil {
			return err
		}
		_, err := relay.ReadMsg(ctx, transfer.SenderRelayAck)
		if err != nil {
			return err
		}

		if len(msgs) > 0 {
			msgs[0] <- transfer.Relay
		}
	} else {
		tc = direct
		// Communicate to the sender that we are doing direct communication.
		if err := relay.WriteMsg(ctx, transfer.Msg{Type: transfer.ReceiverDirectCommunication}); err != nil {
			return err
		}

		// Tell rendezvous server that we can close the connection.
		if err := rc.WriteMsg(ctx, rendezvous.Msg{Type: rendezvous.ReceiverToRendezvousClose}); err != nil {
			return err
		}

		if len(msgs) > 0 {
			msgs[0] <- transfer.Direct
		}
	}

	// Request the payload and receive it.
	if tc.WriteMsg(ctx, transfer.Msg{Type: transfer.ReceiverRequestPayload}) != nil {
		return err
	}
	if err := receivePayload(ctx, tc, dst, msgs...); err != nil {
		return err
	}

	// Closing handshake.
	if err := tc.WriteMsg(ctx, transfer.Msg{Type: transfer.ReceiverPayloadAck}); err != nil {
		return err
	}

	_, err = tc.ReadMsg(ctx, transfer.SenderClosing)

	if err != nil {
		return err
	}

	if err := tc.WriteMsg(ctx, transfer.Msg{Type: transfer.ReceiverClosingAck}); err != nil {
		return err
	}

	// Tell rendezvous to close connection.
	if err := rc.WriteMsg(ctx, rendezvous.Msg{Type: rendezvous.ReceiverToRendezvousClose}); err != nil {
		return err
	}
	return nil
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

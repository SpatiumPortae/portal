//go:build js

package receiver

import (
	"context"
	"io"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/protocol/rendezvous"
	"github.com/SpatiumPortae/portal/protocol/transfer"
)

// doReceive performs the transfer protocol on the receiving end.
// This function is only built for the js platform.
func doReceive(ctx context.Context, relayTc conn.Transfer, addr string, dst io.Writer, msgs ...chan interface{}) error {
	// Communicate to the sender that we are using relay transfer.
	if err := relayTc.WriteMsg(ctx, transfer.Msg{Type: transfer.ReceiverRelayCommunication}); err != nil {
		return err
	}
	_, err := relayTc.ReadMsg(ctx, transfer.SenderRelayAck)
	if err != nil {
		return err
	}

	if len(msgs) > 0 {
		msgs[0] <- transfer.Relay
	}

	// Request the payload and receive it.
	if relayTc.WriteMsg(ctx, transfer.Msg{Type: transfer.ReceiverRequestPayload}) != nil {
		return err
	}
	if err := receivePayload(ctx, relayTc, dst, msgs...); err != nil {
		return err
	}

	// Closing handshake.
	if err := relayTc.WriteMsg(ctx, transfer.Msg{Type: transfer.ReceiverPayloadAck}); err != nil {
		return err
	}

	_, err = relayTc.ReadMsg(ctx, transfer.SenderClosing)

	if err != nil {
		return err
	}

	if err := relayTc.WriteMsg(ctx, transfer.Msg{Type: transfer.ReceiverClosingAck}); err != nil {
		return err
	}

	// Retrieve a unencrypted channel to rendezvous.
	rc := conn.Rendezvous{Conn: relayTc.Conn}
	if err := rc.WriteMsg(ctx, rendezvous.Msg{Type: rendezvous.ReceiverToRendezvousClose}); err != nil {
		return err
	}
	return nil
}

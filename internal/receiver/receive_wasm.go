//go:build js

package receiver

import (
	"io"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/protocol/rendezvous"
	"github.com/SpatiumPortae/portal/protocol/transfer"
)

// doReceive performs the transfer protocol on the receiving end.
// This function is only built for the js platform.
func doReceive(relay conn.Transfer, dst io.Writer, writers ...io.Writer) (Receiver,error) {
	if err := relay.WriteMsg(transfer.Msg{Type: transfer.ReceiverHandshake}); err != nil {
		return nil, err
	}
	msg, err := relay.ReadMsg(transfer.SenderHandshake)
	if err != nil {
		return nil, err
	}
	// Retrieve a unencrypted channel to rendezvous.
	rc := conn.Rendezvous{Conn: relay.Conn}

	// Communicate to the sender that we are using relay transfer.
	if err := relay.WriteMsg(transfer.Msg{Type: transfer.ReceiverRelayCommunication}); err != nil {
		return err
	}
	_, err := relay.ReadMsg(transfer.SenderRelayAck)
	if err != nil {
		return err
	}
	return receiver{
		transferType: transfer.Relay,
		payloadSize:  msg.Payload.PayloadSize,
		tc:           relay,
		rc:           rc,
		dst:          dst,
		writers:      writers
  }
	return nil
}

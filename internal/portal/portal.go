package portal

import (
	"context"
	"io"

	"github.com/SpatiumPortae/portal/internal/receiver"
	"github.com/SpatiumPortae/portal/internal/sender"
)

// Send executes the portal send sequence. The initial connection with the relay
// server is performed synchronously, after that the transfer sequence is performed
// asynchronously. The function returns a portal password, a error from the rendezvous
// initial rendezvous connection, and a channel on which errors from the transfer sequence
// can be listened to. The provided config will be merged with the default config.
func Send(ctx context.Context, payload io.Reader, payloadSize int64, config *Config) (string, error, chan error) {
	merged := MergeConfig(defaultConfig, config)
	errC := make(chan error, 1) // buffer channel as to not block send.
	rc, password, err := sender.ConnectRendezvous(ctx, merged.RendezvousAddr)
	if err != nil {
		return "", err, nil
	}
	go func() {
		defer close(errC)
		tc, err := sender.SecureConnection(ctx, rc, password)
		if err != nil {
			errC <- err
			return
		}
		if err := sender.Transfer(ctx, tc, payload, payloadSize); err != nil {
			errC <- err
			return
		}
	}()
	return password, nil, errC
}

// Receive executes the portal receive sequence. The payload is written
// to the provided writer. The provided config will be merged with the
// default config.
func Receive(ctx context.Context, dst io.Writer, password string, config *Config) error {
	merged := MergeConfig(defaultConfig, config)
	rc, err := receiver.ConnectRendezvous(merged.RendezvousAddr)
	if err != nil {
		return err
	}
	tc, err := receiver.SecureConnection(ctx, rc, password)
	if err != nil {
		return err
	}
	if err := receiver.Receive(ctx, tc, dst); err != nil {
		return err
	}
	return nil
}

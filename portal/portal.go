package portal

import (
	"io"

	"github.com/SpatiumPortae/portal/internal/receiver"
	"github.com/SpatiumPortae/portal/internal/sender"
)

// Send executes the portal send sequence. The intial connection with the Rendezous
// server is performed synchronously, after that the transfer sequence is performed
// asynchronously. The function returns a portal password, a error from the rendezvous
// intial rendezvous connection, and a channel on which errors from the transfer sequence
// can be listend to. The provided config will be merged with the default config.
func Send(payload io.Reader, payloadSize int64, config *Config) (string, error, chan error) {
	merged := MergeConfig(defaultConfig, config)
	if err := sender.Init(); err != nil {
		return "", err, nil
	}
	errC := make(chan error, 1) // buffer channel as to not block send.
	rc, password, err := sender.ConnectRendezvous(merged.RendezvousAddr)
	if err != nil {
		return "", err, nil
	}
	go func() {
		defer close(errC)
		tc, err := sender.SecureConnection(rc, password)
		if err != nil {
			errC <- err
			return
		}
		if err := sender.Transfer(tc, payload, payloadSize); err != nil {
			errC <- err
			return
		}
	}()
	return password, nil, errC
}

// Receive executes the portal receive sequence. The payload is written
// to the provided writer. The provided config will be merged with the
// default config.
func Receive(dst io.Writer, password string, config *Config) error {
	merged := MergeConfig(defaultConfig, config)
	rc, err := receiver.ConnectRendezvous(merged.RendezvousAddr)
	if err != nil {
		return err
	}
	tc, err := receiver.SecureConnection(rc, password)
	if err != nil {
		return err
	}
	if err := receiver.Receive(tc, dst); err != nil {
		return err
	}
	return nil
}

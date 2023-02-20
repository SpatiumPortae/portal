// handlers.go sepcifies the websocket handlers that rendezvous server uses to facilitate communcation between sender and receiver.
package rendezvous

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/logger"
	"github.com/SpatiumPortae/portal/protocol/rendezvous"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

// ------------------------------------------------------ Handlers -----------------------------------------------------

// handleEstablishSender returns a websocket handler that communicates with the sender.
func (s *Server) handleEstablishSender() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger, err := logger.FromContext(ctx)
		if err != nil {
			return
		}
		c, err := conn.FromContext(ctx)
		if err != nil {
			logger.Error("getting Conn from request context", zap.Error(err))
			return
		}
		rc := conn.Rendezvous{Conn: c}
		logger.Info("sender connected")

		id := s.ids.Bind()
		logger = logger.With(zap.Int("id", id))
		logger.Info("bound id")
		defer func() {
			s.ids.Delete(id)
			logger.Info("freed id")
		}()

		err = rc.WriteMsg(ctx, rendezvous.Msg{
			Type: rendezvous.RendezvousToSenderBind,
			Payload: rendezvous.Payload{
				ID: id,
			},
		})
		if err != nil {
			logger.Error("binding communcation ID", zap.Error(err))
			return
		}

		msg, err := rc.ReadMsg(ctx, rendezvous.SenderToRendezvousEstablish)
		if err != nil {
			logger.Error("establishing sender", zap.Error(err))
			return
		}

		// Allocate a mailbox for this communication.
		mailbox := &Mailbox{
			Sender:   make(chan []byte),
			Receiver: make(chan []byte),
		}
		s.mailboxes.StoreMailbox(msg.Payload.Password, mailbox)
		password := msg.Payload.Password

		// wait for receiver to connect or connection timeout
		timeout := time.NewTimer(RECEIVER_CONNECT_TIMEOUT)
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				logger.Error("context error while waiting for receiver", zap.Error(ctx.Err()))
			}
			logger.Info("closing handler")
			return
		case <-timeout.C:
			logger.Warn("waiting for receiver timed out")
			return
		case <-mailbox.Sender:
			break
		}

		err = rc.WriteMsg(ctx, rendezvous.Msg{
			Type: rendezvous.RendezvousToSenderReady,
		})

		if err != nil {
			logger.Error("sending ready message to sender", zap.Error(err))
			return
		}

		msg, err = rc.ReadMsg(ctx, rendezvous.SenderToRendezvousPAKE)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("performing PAKE exchange", zap.Error(err))
			return
		}
		// send PAKE bytes to receiver
		mailbox.Receiver <- msg.Payload.Bytes
		// respond with receiver PAKE bytes
		err = rc.WriteMsg(ctx, rendezvous.Msg{
			Type: rendezvous.RendezvousToSenderPAKE,
			Payload: rendezvous.Payload{
				Bytes: <-mailbox.Sender,
			},
		})
		if err != nil {
			logger.Error("sending PAKE bytes to sender", zap.Error(err))
			return
		}

		msg, err = rc.ReadMsg(ctx, rendezvous.SenderToRendezvousSalt)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("performing salt exchange", zap.Error(err))
			return
		}

		// Send the salt to the receiver.
		mailbox.Receiver <- msg.Payload.Salt
		// Start forwarder and relay
		forward := make(chan []byte)
		wg := sync.WaitGroup{}
		relayCtx, cancel := context.WithCancel(ctx)

		wg.Add(2)
		go s.forwarder(relayCtx, &wg, rc, forward, logger)
		s.relay(relayCtx, &wg, rc, forward, mailbox.Sender, mailbox.Receiver, logger)

		// We want to make sure that the both forwarder and relay have terminated
		cancel()
		wg.Wait()

		// Deallocate mailbox
		logger.Info("deallocating mailbox")
		s.mailboxes.Delete(password)
		logger.Info("sender closing")
	}
}

// handleEstablishReceiver returns a websocket handler that communicates with the sender.
func (s *Server) handleEstablishReceiver() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger, err := logger.FromContext(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		c, err := conn.FromContext(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Error("getting Conn from request context", zap.Error(err))
			return
		}
		rc := conn.Rendezvous{Conn: c}
		logger.Info("receiver connected")

		// Establish receiver.
		msg, err := rc.ReadMsg(ctx, rendezvous.ReceiverToRendezvousEstablish)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("establishing receiver", zap.Error(err))
			return
		}

		mailbox, err := s.mailboxes.GetMailbox(msg.Payload.Password)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Error("failed to get mailbox", zap.Error(err))
			return
		}
		if mailbox.hasReceiver {
			w.WriteHeader(http.StatusBadRequest)
			logger.Warn("mailbox already have a receiver")
			return
		}
		// this receiver was first, reserve this mailbox for it to receive
		mailbox.hasReceiver = true
		s.mailboxes.StoreMailbox(msg.Payload.Password, mailbox)

		// notify sender we are connected
		mailbox.Sender <- []byte{}
		// send back received sender PAKE bytes
		err = rc.WriteMsg(ctx, rendezvous.Msg{
			Type: rendezvous.RendezvousToReceiverPAKE,
			Payload: rendezvous.Payload{
				Bytes: <-mailbox.Receiver,
			},
		})
		if err != nil {
			logger.Error("sending PAKE bytes to receiver", zap.Error(err))
			return
		}

		msg, err = rc.ReadMsg(ctx, rendezvous.ReceiverToRendezvousPAKE)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("performing PAKE exchange", zap.Error(err))
			return
		}

		mailbox.Sender <- msg.Payload.Bytes
		err = rc.WriteMsg(ctx, rendezvous.Msg{
			Type: rendezvous.RendezvousToReceiverSalt,
			Payload: rendezvous.Payload{
				Salt: <-mailbox.Receiver,
			},
		})
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("exchanging salt", zap.Error(err))
		}

		// Start forwarder and relay
		forward := make(chan []byte)
		wg := sync.WaitGroup{}
		subCtx, cancel := context.WithCancel(ctx)

		wg.Add(2)
		go s.forwarder(subCtx, &wg, rc, forward, logger)
		s.relay(subCtx, &wg, rc, forward, mailbox.Receiver, mailbox.Sender, logger)
		cancel()

		wg.Wait()

		logger.Info("receiver closing")
	}
}

//nolint:errcheck
func (s *Server) ping() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	}
}

//nolint:errcheck
func (s *Server) handleVersionCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger, err := logger.FromContext(ctx)
		if err != nil {
			return
		}

		res, err := http.Get("https://api.github.com/repos/SpatiumPortae/portal/releases?per_page=1")
		if err != nil {
			logger.Error("fetching latest version tag from GitHub releases API", zap.Error(err))
			return
		}
		defer res.Body.Close()

		io.Copy(w, res.Body)
	}
}

// ------------------------------------------------------ Helpers ------------------------------------------------------

// forwarder reads from the connection and forwards the message to the provided channel.
// Transient errors are logged on the provided logger.
func (s *Server) forwarder(ctx context.Context, wg *sync.WaitGroup, rc conn.Rendezvous, forward chan<- []byte, logger *zap.Logger) {
	forwardLogger := logger.With(zap.String("component", "forwarder"))
	forwardLogger.Info("starting forwarder")
	defer wg.Done()
	defer close(forward)
	for {
		payload, err := rc.ReadRaw(ctx)
		switch {
		case errors.Is(err, io.EOF):
			forwardLogger.Error("connection forcefully closed", zap.Error(err))
			return

		// TODO: Extract closure status out to the Conn implementation
		//  Would be better to return a custom error, so we are not
		//  as heavily coupled with the websocket library

		case websocket.CloseStatus(err) == websocket.StatusNormalClosure:
			forwardLogger.Info("connection closed, closing forwarder")
			return
		case errors.Is(err, context.Canceled):
			forwardLogger.Info("context canceled, closing forwarder")
			return
		case err != nil:
			forwardLogger.Error("error reading from connection, closing forwarder", zap.Error(err))
			return
		}

		var msg rendezvous.Msg
		if err := json.Unmarshal(payload, &msg); err == nil {
			logger.Info("received unencrypted message, closing forwarder")
			return
		}
		forward <- payload
	}
}

func (s *Server) relay(ctx context.Context, wg *sync.WaitGroup, rc conn.Rendezvous, forward, relayIn <-chan []byte, relayOut chan<- []byte, logger *zap.Logger) {
	relayLogger := logger.With(zap.String("component", "relay"))
	relayLogger.Info("starting")
	defer wg.Done()
	defer close(relayOut)
	for {
		select {
		case <-ctx.Done():
			relayLogger.Info("received context done signal")
			return
		case forwarded, more := <-forward:
			if !more {
				relayLogger.Info("forwarding channel closed, closing relay")
				return
			}
			relayOut <- forwarded
		case relayed, more := <-relayIn:
			if !more {
				relayLogger.Info("relay channel closed, closing relay")
				return
			}
			if err := rc.WriteRaw(ctx, relayed); err != nil {
				relayLogger.Error("writing relayed message to connection")
				return
			}
		}
	}
}

// handlers.go sepcifies the websocket handlers that rendezvous server uses to facilitate communcation between sender and receiver.
package rendezvous

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/logger"
	"github.com/SpatiumPortae/portal/protocol/rendezvous"
	"go.uber.org/zap"
)

// handleEstablishSender returns a websocket handler that communicates with the sender.
func (s *Server) handleEstablishSender() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger, err := logger.FromContext(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		c, err := conn.FromContext(ctx)
		if err != nil {
			logger.Error("getting Conn from request context", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		rc := conn.Rendezvous{Conn: c}
		logger.Info("sender connected")
		// Bind an ID to this communication and send to the sender
		id := s.ids.Bind()
		defer func() {
			s.ids.Delete(id)
			logger.Info("freed id", zap.Int("id", id))
		}()
		err = rc.WriteMsg(rendezvous.Msg{
			Type: rendezvous.RendezvousToSenderBind,
			Payload: rendezvous.Payload{
				ID: id,
			},
		})
		logger.Info("bound id", zap.Int("id", id))
		if err != nil {
			logger.Error("binding communcation ID", zap.Error(err))
			return
		}

		msg, err := rc.ReadMsg(rendezvous.SenderToRendezvousEstablish)
		if err != nil {
			logger.Error("establishing sender", zap.Error(err))
			return
		}

		// Allocate a mailbox for this communication.
		mailbox := &Mailbox{
			CommunicationChannel: make(chan []byte),
			Quit:                 make(chan bool),
		}
		s.mailboxes.StoreMailbox(msg.Payload.Password, mailbox)
		password := msg.Payload.Password

		// wait for receiver to connect or connection timeout
		timeout := time.NewTimer(RECEIVER_CONNECT_TIMEOUT)
		select {
		case <-timeout.C:
			logger.Warn("waiting for receiver timeout")
			return
		case <-mailbox.CommunicationChannel:
			break
		}

		err = rc.WriteMsg(rendezvous.Msg{
			Type: rendezvous.RendezvousToSenderReady,
		})

		if err != nil {
			logger.Error("sending ready message to sender", zap.Error(err))
			return
		}

		msg, err = rc.ReadMsg(rendezvous.SenderToRendezvousPAKE)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("performing PAKE exchange", zap.Error(err))
			return
		}
		// send PAKE bytes to receiver
		mailbox.CommunicationChannel <- msg.Payload.Bytes
		// respond with receiver PAKE bytes
		err = rc.WriteMsg(rendezvous.Msg{
			Type: rendezvous.RendezvousToSenderPAKE,
			Payload: rendezvous.Payload{
				Bytes: <-mailbox.CommunicationChannel,
			},
		})
		if err != nil {
			logger.Error("sending PAKE bytes to sender", zap.Error(err))
			return
		}

		msg, err = rc.ReadMsg(rendezvous.SenderToRendezvousSalt)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("performing salt exchange", zap.Error(err))
			return
		}

		// Send the salt to the receiver.
		mailbox.CommunicationChannel <- msg.Payload.Salt
		// Start the relay of messages between the sender and receiver handlers.
		logger.Info("starting relay service")
		startRelay(s, rc, mailbox, password, logger)
	}
}

// handleEstablishReceiver returns a websocket handler that communicates with the sender.
func (s *Server) handleEstablishReceiver() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger, err := logger.FromContext(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		c, err := conn.FromContext(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Error("getting Conn from request context", zap.Error(err))
			return
		}
		rc := conn.Rendezvous{Conn: c}
		logger.Info("receiver connected")

		// Establish receiver.
		msg, err := rc.ReadMsg(rendezvous.ReceiverToRendezvousEstablish)
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
		password := msg.Payload.Password

		// notify sender we are connected
		mailbox.CommunicationChannel <- []byte{}
		// send back received sender PAKE bytes
		err = rc.WriteMsg(rendezvous.Msg{
			Type: rendezvous.RendezvousToReceiverPAKE,
			Payload: rendezvous.Payload{
				Bytes: <-mailbox.CommunicationChannel,
			},
		})
		if err != nil {
			logger.Error("sending PAKE bytes to receiver", zap.Error(err))
			return
		}

		msg, err = rc.ReadMsg(rendezvous.ReceiverToRendezvousPAKE)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("performing PAKE exchange", zap.Error(err))
			return
		}

		mailbox.CommunicationChannel <- msg.Payload.Bytes
		err = rc.WriteMsg(rendezvous.Msg{
			Type: rendezvous.RendezvousToReceiverSalt,
			Payload: rendezvous.Payload{
				Salt: <-mailbox.CommunicationChannel,
			},
		})
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("exchanging salt", zap.Error(err))
		}

		logger.Info("start relay service")
		startRelay(s, rc, mailbox, password, logger)
	}
}

// starts the relay service, closing it on request (if i.e. clients can communicate directly)
func startRelay(s *Server, conn conn.Rendezvous, mailbox *Mailbox, mailboxPassword string, logger *zap.Logger) {
	relayForwardCh := make(chan []byte)
	// listen for incoming websocket messages from currently handled client
	go func() {
		for {
			// read raw bytes and pass them on
			payload, err := conn.ReadBytes()
			if err != nil {
				logger.Error("listening to incoming client messages", zap.Error(err))
				mailbox.Quit <- true
				return
			}
			relayForwardCh <- payload
		}
	}()

	for {
		select {
		// received payload from __other client__, relay it to our currently handled client
		case relayReceivePayload := <-mailbox.CommunicationChannel:
			err := conn.WriteBytes(relayReceivePayload) // send raw binary data
			if err != nil {
				logger.Error("relaying bytes, closing relay service", zap.Error(err))
				// close the relay service if writing failed
				mailbox.Quit <- true
				return
			}

		// received payload from __currently handled__ client, relay it to other client
		case relayForwardPayload := <-relayForwardCh:
			var msg rendezvous.Msg
			err := json.Unmarshal(relayForwardPayload, &msg)
			// failed to unmarshal, we are in (encrypted) relay-mode, forward message directly to client
			if err != nil {
				mailbox.CommunicationChannel <- relayForwardPayload
			} else {
				logger.Info("closing relay service")
				// close the relay service if sender requested it
				mailbox.Quit <- true
				return
			}

		// deallocate mailbox and quit
		case <-mailbox.Quit:
			s.mailboxes.Delete(mailboxPassword)
			return
		}
	}
}

//nolint:errcheck
func (s *Server) ping() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	}
}

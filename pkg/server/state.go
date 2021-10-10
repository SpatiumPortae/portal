package server

type SenderState = int

const (
	AwaitingSenderConnection SenderState = iota
	AwaitingReceiverRequests
)

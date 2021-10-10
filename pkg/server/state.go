package server

type SenderState = int

const (
	AwaitingSender SenderState = iota
	AwaitingReceiverRequests
)

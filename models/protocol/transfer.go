package protocol

type TransferMessageType int

const (
	TransferError TransferMessageType = iota
	ReceiverHandshake
	SenderHandshake
	ReceiverRequestPayload
	SenderPayloadSent
	ReceiverAckPayload
	ReceiverClosing
	SenderClosing
	ReceiverClosingAck
)

type TransferMessage struct {
	Type    TransferMessageType `json:"msg_type"`
	Message string              `json:"msg"`
}

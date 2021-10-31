package protocol

type TransferMessageType int

const (
	TransferError TransferMessageType = iota
	ReceiverHandshake
	SenderHandshake
	ReceiverRequestPayload
	SenderPayloadSent
	ReceiverAckPayload
	SenderClosing
	ReceiverClosingAck
)

type TransferMessage struct {
	Type    TransferMessageType `json:"msg_type"`
	Payload interface{}         `json:"payload"`
}

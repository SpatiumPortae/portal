package protocol

type TransferMessageType int

const (
	Error TransferMessageType = iota
	ReceiverHandshake
	SenderHandshake
	ReceiverClosing
	SenderClosing
	ReceiverRequestPayload
	ReceiverAckPayload
	ReceiverClosingAck
)

type TransferMessage struct {
	Type    TransferMessageType `json:"msg_type"`
	Message string              `json:"msg"`
}

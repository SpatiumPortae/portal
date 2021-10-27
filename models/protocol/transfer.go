package protocol

type TransferMessageType int

const (
	Error TransferMessageType = iota
	ClientHandshake
	ServerHandshake
	ClientClosing
	ServerClosing
	ClientRequestPayload
	ClientAckPayload
)

type TransferMessage struct {
	Type    TransferMessageType `json:"msg_type"`
	Message string              `json:"msg"`
}

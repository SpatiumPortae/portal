// Transfer.go specifies the required messaging for the transfer protocol.
package portal

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

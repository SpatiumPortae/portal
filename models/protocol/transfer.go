// transfer.go specifies the necessary messaging needed for the transfer protocol.
package protocol

// TransferMessageType specifies the message type for the messages in the transfer protocol.
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

// TransferMessage specifies a message in the transfer protocol.
type TransferMessage struct {
	Type    TransferMessageType `json:"msg_type"`
	Payload interface{}         `json:"payload"`
}

// SenderHandshakePayload specifies a payload type for announcing the payload size.
type SenderHandshakePayload struct {
	PayloadSize int `json:"payload_size"`
}

//NOTE: should probably implment JSON object for regular string messages as well.

package sender

type TransferState int

const (
	Initial TransferState = iota
	WaitForHandShake
	WaitForFileRequest
	SendingData
	WaitForFileAck
	WaitForCloseMessage
	WaitForCloseAck
	Closing
)

type UIUpdate struct {
	State    TransferState
	Progress float32
}

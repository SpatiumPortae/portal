package receiver

type TransferState int

const (
	Initial TransferState = iota
	RequestingFile
	ReceivingData
	WaitForFileAck
	WaitForCloseAck
)

type UIUpdate struct {
	State    TransferState
	Progress float32
}

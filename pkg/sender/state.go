package sender

type TransferSenderState int

const (
	Initial TransferSenderState = iota
	WaitForHandShake
	WaitForFileRequest
	WaitForFileAck
	WaitForCloseMessage
	WaitForCloseAck
)

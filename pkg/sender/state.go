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
	Progress int
}

func updateUI(ui chan<- UIUpdate, state TransferState, progress ...int) {
	var p int
	if len(progress) > 0 {
		p = progress[0]
	}
	if ui != nil {
		ui <- UIUpdate{State: state, Progress: p}
	}
}

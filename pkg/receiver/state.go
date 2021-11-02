package receiver

type TransferState int

type UIUpdate struct {
	State    TransferState
	Progress float32
}

package ui

type FileInfoMsg struct {
	FileNames []string
	Bytes     int64
}

type ErrorMsg struct {
	Message string
}

type ProgressMsg struct {
	Progress float32
}

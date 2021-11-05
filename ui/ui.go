package ui

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

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

func GracefulUIQuit(uiProgram *tea.Program) {
	time.Sleep(SHUTDOWN_PERIOD)
	uiProgram.Quit()
	fmt.Println("") // hack to persist the last line after ui quit
	time.Sleep(SHUTDOWN_PERIOD)
	os.Exit(0)
}

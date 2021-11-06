package ui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type UIUpdate struct {
	Progress float32
}

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

type FinishedMsg struct {
	Files       []string
	PayloadSize int64
}

var WaitingSpinner = spinner.Spinner{
	Frames: []string{"⠋ ", "⠙ ", "⠹ ", "⠸ ", "⠼ ", "⠴ ", "⠦ ", "⠧ ", "⠇ ", "⠏ "},
	FPS:    time.Second / 12,
}

var CompressingSpinner = spinner.Spinner{
	Frames: []string{"┉┉┉", "┅┅┅", "┄┄┄", "┉ ┉", "┅ ┅", "┄ ┄", " ┉ ", " ┉ ", " ┅ ", " ┅ ", " ┄ "},
	FPS:    time.Second / 3,
}

var TransferSpinner = spinner.Spinner{
	Frames: []string{"»  ", "»» ", "»»»", "   "},
	FPS:    time.Millisecond * 400,
}

var ReceivingSpinner = spinner.Spinner{
	Frames: []string{"   ", "  «", " ««", "«««"},
	FPS:    time.Second / 2,
}

func TopLevelFilesText(fileNames []string) string {
	// parse top level file names and attach number of subfiles in them
	topLevelFileChildren := make(map[string]int)
	for _, f := range fileNames {
		fileTopPath := strings.Split(f, "/")[0]
		subfileCount, wasPresent := topLevelFileChildren[fileTopPath]
		if wasPresent {
			topLevelFileChildren[fileTopPath] = subfileCount + 1
		} else {
			topLevelFileChildren[fileTopPath] = 0
		}
	}
	// read map into formatted strings
	var topLevelFilesText []string
	for fileName, subFileCount := range topLevelFileChildren {
		formattedFileName := fileName
		if subFileCount > 0 {
			formattedFileName = fmt.Sprintf("%s (%d subfiles)", fileName, subFileCount)
		}
		topLevelFilesText = append(topLevelFilesText, formattedFileName)
	}
	sort.Strings(topLevelFilesText)
	return strings.Join(topLevelFilesText, ", ")
}

func GracefulUIQuit(uiProgram *tea.Program) {
	time.Sleep(SHUTDOWN_PERIOD)
	uiProgram.Quit()
	fmt.Println("") // hack to persist the last line after ui quit
	time.Sleep(SHUTDOWN_PERIOD)
	os.Exit(0)
}

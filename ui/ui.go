package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/protocol/transfer"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// -------------------- SHARED UI MESSAGES --------------------

type ErrorMsg error

type ProgressMsg int

type SecureMsg struct {
	Conn conn.Transfer
}
type TransferTypeMsg struct {
	Type transfer.Type
}

// -------------------- SPINNERS -------------------------------

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

// -------------------- SHARED HELPERS ---------------------------

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

// Credits to (legendary Mr. Nilsson): https://yourbasic.org/golang/formatting-byte-size-to-human-readable-format/
func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

// -------------------- SHARED COMMANDS ---------------------------

func QuitCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(SHUTDOWN_PERIOD)
		return tea.Quit()
	}
}

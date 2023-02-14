package ui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/semver"
	"github.com/SpatiumPortae/portal/protocol/transfer"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ------------------------------------------------- Shared UI Messages ------------------------------------------------

type ErrorMsg error

type ProgressMsg int

type SecureMsg struct {
	Conn conn.Transfer
}
type TransferTypeMsg struct {
	Type transfer.Type
}

type TransferStateMessage struct {
	State transfer.MsgType
}

type VersionMsg struct {
	Latest semver.Version
}

// ------------------------------------------------------ Spinners -----------------------------------------------------

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

// --------------------------------------------------- Shared Helpers --------------------------------------------------

func LogSeparator(width int) string {
	paddedWidth := math.Max(0, float64(width)-2*PADDING)
	return fmt.Sprintf("%s\n\n",
		baseStyle.Copy().
			Foreground(lipgloss.Color(SECONDARY_COLOR)).
			Render(strings.Repeat("─", int(math.Min(MAX_WIDTH, paddedWidth)))))
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

// -------------------------------------------------- Shared Commands --------------------------------------------------

func TaskCmd(task string, cmds ...tea.Cmd) tea.Cmd {
	msg := PadText + fmt.Sprintf("• %s", task)
	return tea.Sequence(tea.Println(msg), tea.Batch(cmds...))
}

func QuitCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(SHUTDOWN_PERIOD)
		return tea.Quit()
	}
}

func VersionCmd(version semver.Version) tea.Cmd {
	return func() tea.Msg {
		latest, err := semver.GetPortalLatest()
		if err != nil {
			return ErrorMsg(err)
		}
		return VersionMsg{
			Latest: latest,
		}
	}
}

func ErrorCmd(err error) tea.Cmd {
	cmd := func() tea.Msg {
		return ErrorMsg(err)
	}
	return TaskCmd(ErrorText(err.Error()), cmd)
}

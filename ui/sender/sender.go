package sender

import (
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	"www.github.com/ZinoKader/portal/internal/conn"
	"www.github.com/ZinoKader/portal/internal/sender"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
	"www.github.com/ZinoKader/portal/ui"
)

const (
	copyPasswordKey = "c"
)

type uiState int

// ui state flows from the top down
const (
	showPasswordWithCopy uiState = iota
	showPassword
	showSendingProgress
	showFinished
	showError
)

type ReadyMsg struct{}

type ConnectMsg struct {
	password string
	conn     conn.Rendezvous
}

type FileReadMsg struct {
	files []*os.File
	size  int64
}

type CompressedMsg struct {
	payload *os.File
	size    int64
}

type model struct {
	state        uiState               // defaults to 0 (showPasswordWithCopy)
	transferType protocol.TransferType // defaults to 0 (Unknown)
	errorMessage string
	readyToSend  bool

	msgs chan interface{}

	rendezvousAddr net.TCPAddr

	password         string
	fileNames        []string
	uncompressedSize int64
	payload          *os.File
	payloadSize      int64

	spinner     spinner.Model
	progressBar progress.Model
}

func New(filenames []string, addr net.TCPAddr) *tea.Program {
	m := model{
		progressBar:    ui.ProgressBar,
		fileNames:      filenames,
		rendezvousAddr: addr,
		msgs:           make(chan interface{}, 10),
	}
	m.resetSpinner()
	return tea.NewProgram(m)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		spinner.Tick,
		readFilesCmd(m.fileNames),
		connectCmd(m.rendezvousAddr))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case FileReadMsg:
		m.uncompressedSize = msg.size
		return m, compressFilesCmd(msg.files)

	case CompressedMsg:
		m.payload = msg.payload
		m.payloadSize = msg.size
		m.readyToSend = true
		m.resetSpinner()
		return m, spinner.Tick

	case ConnectMsg:
		m.password = msg.password
		return m, secureCmd(msg.conn, msg.password)

	case ui.TransferTypeMsg:
		m.transferType = msg.Type
		return m, listenTransferCmd(m.msgs)

	case ui.SecureMsg:
		// In the case we are not ready to send yet we pass on the same message.
		if !m.readyToSend {
			return m, func() tea.Msg {
				return msg
			}
		}
		cmds := []tea.Cmd{
			transferCmd(msg.Conn, m.payload, m.payloadSize, m.msgs),
			listenTransferCmd(m.msgs),
		}

		return m, tea.Batch(cmds...)

	case ui.ProgressMsg:
		cmds := []tea.Cmd{listenTransferCmd(m.msgs)}
		if m.state != showSendingProgress {
			m.state = showSendingProgress
			m.resetSpinner()
			cmds = append(cmds, spinner.Tick)
		}
		percent := float64(msg) / float64(m.payloadSize)
		if percent > 1.0 {
			percent = 1.0
		}
		cmds = append(cmds, m.progressBar.SetPercent(percent))
		return m, tea.Batch(cmds...)

	case ui.FinishedMsg:
		m.state = showFinished
		return m, ui.QuitCmd()

	case ui.ErrorMsg:
		m.state = showError
		m.errorMessage = msg.Error()
		return m, nil

	case tea.KeyMsg:
		if m.state == showPasswordWithCopy && strings.ToLower(msg.String()) == copyPasswordKey {
			m.state = showPassword
			clipboard.WriteAll(fmt.Sprintf("portal receive %s", m.password))
			return m, nil
		}
		if tools.Contains(ui.QuitKeys, strings.ToLower(msg.String())) {
			return m, tea.Quit
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.progressBar.Width = msg.Width - 2*ui.PADDING - 4
		if m.progressBar.Width > ui.MAX_WIDTH {
			m.progressBar.Width = ui.MAX_WIDTH
		}
		return m, nil

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progressBar.Update(msg)
		m.progressBar = progressModel.(progress.Model)
		return m, cmd

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m model) View() string {
	// Setup strings to use in view.

	uncompressed := ui.BoldText(tools.ByteCountSI(m.uncompressedSize))
	readiness := fmt.Sprintf("%s Compressing objects (%s), preparing to send", m.spinner.View(), uncompressed)
	if m.readyToSend {
		readiness = fmt.Sprintf("%s Awaiting receiver, ready to send", m.spinner.View())
	}
	if m.state == showSendingProgress {
		readiness = fmt.Sprintf("%s Sending", m.spinner.View())
	}

	sort.Strings(m.fileNames)
	filesToSend := ui.ItalicText(strings.Join(m.fileNames, ", "))

	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%s %d object", readiness, len(m.fileNames)))
	if len(m.fileNames) > 1 {
		builder.WriteRune('s')
	}
	if m.payloadSize != 0 {
		compressed := ui.BoldText(tools.ByteCountSI(m.payloadSize))
		builder.WriteString(fmt.Sprintf(" (%s)", compressed))
	}

	switch m.transferType {
	case protocol.Direct:
		builder.WriteString(" directly to receiver")
	case protocol.Relay:
		builder.WriteString(" to receiver using relay")
	case protocol.Unknown:
	}

	indentedWrappedFiles := indent.String(wordwrap.String(fmt.Sprintf("Sending: %s", filesToSend), ui.MAX_WIDTH), ui.PADDING)
	builder.WriteString("\n\n")
	builder.WriteString(indentedWrappedFiles)
	fileInfoText := builder.String()

	switch m.state {

	case showPassword, showPasswordWithCopy:

		copyText := "(password copied to clipboard)"
		if m.state == showPasswordWithCopy {
			copyText = "(press 'c' to copy the command to your clipboard)"
		}
		return "\n" +
			ui.PadText + ui.InfoStyle(fileInfoText) + "\n\n" +
			ui.PadText + ui.InfoStyle("On the receiving end, run:") + "\n" +
			ui.PadText + ui.InfoStyle(fmt.Sprintf("portal receive %s", m.password)) + "\n\n" +
			ui.PadText + ui.HelpStyle(copyText) + "\n\n"

	case showSendingProgress:
		return "\n" +
			ui.PadText + ui.InfoStyle(fileInfoText) + "\n\n" +
			ui.PadText + m.progressBar.View() + "\n\n" +
			ui.PadText + ui.QuitCommandsHelpText + "\n\n"

	case showFinished:
		indentedWrappedFiles := indent.String(fmt.Sprintf("Sent: %s", wordwrap.String(ui.ItalicText(ui.TopLevelFilesText(m.fileNames)), ui.MAX_WIDTH)), ui.PADDING)
		finishedText := fmt.Sprintf("Sent %d objects (%s compressed)\n\n%s", len(m.fileNames), tools.ByteCountSI(m.payloadSize), indentedWrappedFiles)
		return "\n" +
			ui.PadText + ui.InfoStyle(finishedText) + "\n\n" +
			ui.PadText + m.progressBar.View() + "\n\n" +
			ui.PadText + ui.QuitCommandsHelpText + "\n\n"

	case showError:
		return m.errorMessage

	default:
		return ""
	}
}

// connectCmd command that connects to the rendezvous server.
func connectCmd(addr net.TCPAddr) tea.Cmd {
	return func() tea.Msg {
		rc, password, err := sender.ConnectRendezvous(addr)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return ConnectMsg{password: password, conn: rc}
	}
}

// secureCmd command that secures a connection for transfer.
func secureCmd(rc conn.Rendezvous, password string) tea.Cmd {
	return func() tea.Msg {
		tc, err := sender.SecureConnection(rc, password)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return ui.SecureMsg{Conn: tc}
	}
}

// transferCmd command that does the transfer sequence.
// The msgs channel is used to provide intermediate messages to the ui.
func transferCmd(tc conn.Transfer, payload io.Reader, payloadSize int64, msgs ...chan interface{}) tea.Cmd {
	return func() tea.Msg {
		err := sender.Transfer(tc, payload, payloadSize, msgs...)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return ui.FinishedMsg{}
	}
}

// readFilesCmd command that reads the files from the provided paths.
func readFilesCmd(paths []string) tea.Cmd {
	return func() tea.Msg {
		files, err := tools.ReadFiles(paths)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		size, err := tools.FilesTotalSize(files)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return FileReadMsg{files: files, size: size}
	}
}

// compressFilesCmd is a command that compresses and archives the
// provided files.
func compressFilesCmd(files []*os.File) tea.Cmd {
	return func() tea.Msg {
		tar, size, err := tools.ArchiveAndCompressFiles(files)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return CompressedMsg{payload: tar, size: size}
	}
}

// listenTransferCmd is a command that listens to the provided
// and channel and formats messages.
func listenTransferCmd(msgs chan interface{}) tea.Cmd {
	return func() tea.Msg {
		msg := <-msgs
		switch v := msg.(type) {
		case protocol.TransferType:
			return ui.TransferTypeMsg{Type: v}
		case int:
			return ui.ProgressMsg(v)
		default:
			return nil
		}
	}
}

func (m *model) resetSpinner() {
	m.spinner = spinner.NewModel()
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ELEMENT_COLOR))
	if m.readyToSend {
		m.spinner.Spinner = ui.WaitingSpinner
	} else {
		m.spinner.Spinner = ui.CompressingSpinner
	}
	if m.state == showSendingProgress {
		m.spinner.Spinner = ui.TransferSpinner
	}
}

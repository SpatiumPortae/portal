package receiver

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	"www.github.com/ZinoKader/portal/internal/conn"
	"www.github.com/ZinoKader/portal/internal/receiver"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
	"www.github.com/ZinoKader/portal/ui"
)

type uiState int

// ui state flows from the top down
const (
	showEstablishing uiState = iota
	showReceivingProgress
	showDecompressing
	showFinished
	showError
)

type connectMsg struct {
	conn conn.Rendezvous
}

type payloadSizeMsg struct {
	size int64
}

type receiveDoneMsg struct {
	temp *os.File
}

type decompressionDoneMsg struct {
	filenames               []string
	decompressedPayloadSize int64
}

type model struct {
	state        uiState
	transferType protocol.TransferType
	password     string

	msgs chan interface{}

	rendezvousAddr net.TCPAddr

	receivedFiles           []string
	payloadSize             int64
	decompressedPayloadSize int64

	spinner      spinner.Model
	progressBar  progress.Model
	errorMessage string
}

func New(addr net.TCPAddr, password string) *tea.Program {
	m := model{
		progressBar:    ui.Progressbar,
		msgs:           make(chan interface{}, 10),
		password:       password,
		rendezvousAddr: addr,
	}
	m.resetSpinner()
	return tea.NewProgram(m)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(spinner.Tick, connectCmd(m.rendezvousAddr))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case connectMsg:
		return m, secureCmd(msg.conn, m.password)

	case ui.SecureMsg:
		return m, tea.Batch(receiveCmd(msg.Conn, m.msgs), listenReceiveCmd(m.msgs))

	case payloadSizeMsg:
		m.payloadSize = msg.size
		return m, listenReceiveCmd(m.msgs)

	case ui.TransferTypeMsg:
		m.transferType = msg.Type
		return m, listenReceiveCmd(m.msgs)

	case ui.ProgressMsg:
		cmds := []tea.Cmd{listenReceiveCmd(m.msgs)}
		if m.state != showReceivingProgress {
			m.state = showReceivingProgress
			m.resetSpinner()
			cmds = append(cmds, spinner.Tick)
		}
		percent := float64(msg) / float64(m.payloadSize)
		if percent > 1.0 {
			percent = 1.0
		}
		cmds = append(cmds, m.progressBar.SetPercent(percent))
		return m, tea.Batch(cmds...)

	case receiveDoneMsg:
		m.state = showDecompressing
		m.resetSpinner()
		cmds := []tea.Cmd{
			spinner.Tick,
			decompressCmd(msg.temp),
		}
		return m, tea.Batch(cmds...)

	case decompressionDoneMsg:
		m.state = showFinished
		m.receivedFiles = msg.filenames
		m.decompressedPayloadSize = msg.decompressedPayloadSize
		return m, ui.QuitCmd()

	case ui.ErrorMsg:
		m.state = showError
		m.errorMessage = msg.Error()
		return m, nil

	case tea.KeyMsg:
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

	switch m.state {

	case showEstablishing:
		return "\n" +
			ui.PadText + ui.InfoStyle(fmt.Sprintf("%s Establishing connection with sender", m.spinner.View())) + "\n\n" +
			ui.PadText + ui.QuitCommandsHelpText + "\n\n"

	case showReceivingProgress:
		var transfer string
		if m.transferType == protocol.Direct {
			transfer = "direct"
		} else {
			transfer = "relay"
		}

		payloadSize := ui.BoldText(tools.ByteCountSI(m.payloadSize))
		receivingText := fmt.Sprintf("%s Receiving files (total size %s) using %s transfer", m.spinner.View(), payloadSize, transfer)
		return "\n" +
			ui.PadText + ui.InfoStyle(receivingText) + "\n\n" +
			ui.PadText + m.progressBar.View() + "\n\n" +
			ui.PadText + ui.QuitCommandsHelpText + "\n\n"

	case showDecompressing:
		payloadSize := ui.BoldText(tools.ByteCountSI(m.payloadSize))
		decompressingText := fmt.Sprintf("%s Decompressing payload (%s compressed) and writing to disk", m.spinner.View(), payloadSize)
		return "\n" +
			ui.PadText + ui.InfoStyle(decompressingText) + "\n\n" +
			ui.PadText + m.progressBar.View() + "\n\n" +
			ui.PadText + ui.QuitCommandsHelpText + "\n\n"

	case showFinished:
		indentedWrappedFiles := indent.String(fmt.Sprintf("Received: %s", wordwrap.String(ui.ItalicText(ui.TopLevelFilesText(m.receivedFiles)), ui.MAX_WIDTH)), ui.PADDING)

		var oneOrMoreFiles string
		if len(m.receivedFiles) > 1 {
			oneOrMoreFiles = "files"
		} else {
			oneOrMoreFiles = "file"
		}
		finishedText := fmt.Sprintf("Received %d %s (%s compressed)\n\n%s", len(m.receivedFiles), oneOrMoreFiles, tools.ByteCountSI(m.payloadSize), indentedWrappedFiles)
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

func connectCmd(addr net.TCPAddr) tea.Cmd {
	return func() tea.Msg {
		rc, err := receiver.ConnectRendezvous(addr)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return connectMsg{conn: rc}
	}
}

func secureCmd(rc conn.Rendezvous, password string) tea.Cmd {
	return func() tea.Msg {
		tc, err := receiver.SecureConnection(rc, password)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return ui.SecureMsg{Conn: tc}
	}
}

func receiveCmd(tc conn.Transfer, msgs ...chan interface{}) tea.Cmd {
	return func() tea.Msg {
		temp, err := os.CreateTemp(os.TempDir(), tools.RECEIVE_TEMP_FILE_NAME_PREFIX)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		if err := receiver.Receive(tc, temp, msgs...); err != nil {
			return ui.ErrorMsg(err)
		}
		return receiveDoneMsg{temp: temp}
	}
}

func listenReceiveCmd(msgs chan interface{}) tea.Cmd {
	return func() tea.Msg {
		msg := <-msgs
		switch v := msg.(type) {
		case protocol.TransferType:
			return ui.TransferTypeMsg{Type: v}
		case int:
			return ui.ProgressMsg(v)
		case int64:
			return payloadSizeMsg{size: v}
		default:
			return nil
		}
	}
}

func decompressCmd(temp *os.File) tea.Cmd {
	return func() tea.Msg {
		// reset file position for reading
		temp.Seek(0, 0)

		filenames, decompressedSize, err := tools.DecompressAndUnarchiveBytes(temp)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return decompressionDoneMsg{filenames: filenames, decompressedPayloadSize: decompressedSize}
	}
}

func (m *model) resetSpinner() {
	m.spinner = spinner.NewModel()
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ELEMENT_COLOR))
	if m.state == showEstablishing {
		m.spinner.Spinner = ui.WaitingSpinner
	}
	if m.state == showDecompressing {
		m.spinner.Spinner = ui.CompressingSpinner
	}
	if m.state == showReceivingProgress {
		m.spinner.Spinner = ui.TransferSpinner
	}
}

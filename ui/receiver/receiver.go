package receiver

import (
	"fmt"
	"os"
	"strings"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/file"
	"github.com/SpatiumPortae/portal/internal/receiver"
	"github.com/SpatiumPortae/portal/internal/semver"
	"github.com/SpatiumPortae/portal/protocol/transfer"
	"github.com/SpatiumPortae/portal/ui"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	"golang.org/x/exp/slices"
)

// -------------------- UI STATE --------------------------------
type uiState int

// Flows from the top down.
const (
	showEstablishing uiState = iota
	showReceivingProgress
	showDecompressing
	showFinished
	showError
)

// -------------------- UI MESSAGES ------------------------------
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

// -------------------- MODEL -------------------------------------

type Option func(m *model)

func WithVersion(version semver.Version) Option {
	return func(m *model) {
		m.version = &version
	}
}

type model struct {
	state        uiState
	transferType transfer.Type
	password     string

	msgs chan interface{}

	rendezvousAddr string

	receivedFiles           []string
	payloadSize             int64
	decompressedPayloadSize int64
	version                 *semver.Version

	spinner      spinner.Model
	progressBar  progress.Model
	errorMessage string
}

// New creates a receiver program.
func New(addr string, password string, opts ...Option) *tea.Program {
	m := model{
		progressBar:    ui.Progressbar,
		msgs:           make(chan interface{}, 10),
		password:       password,
		rendezvousAddr: addr,
	}
	for i := range opts {
		opts[i](&m)
	}
	m.resetSpinner()
	return tea.NewProgram(m)
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{spinner.Tick, connectCmd(m.rendezvousAddr)}
	if m.version != nil {
		cmds = append(cmds, ui.VersionCmd(*m.version))
	}
	return tea.Batch(cmds...)
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
		if slices.Contains(ui.QuitKeys, strings.ToLower(msg.String())) {
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
		var transferType string
		if m.transferType == transfer.Direct {
			transferType = "direct"
		} else {
			transferType = "relay"
		}

		payloadSize := ui.BoldText(ui.ByteCountSI(m.payloadSize))
		receivingText := fmt.Sprintf("%s Receiving files (total size %s) using %s transfer", m.spinner.View(), payloadSize, transferType)
		return "\n" +
			ui.PadText + ui.InfoStyle(receivingText) + "\n\n" +
			ui.PadText + m.progressBar.View() + "\n\n" +
			ui.PadText + ui.QuitCommandsHelpText + "\n\n"

	case showDecompressing:
		payloadSize := ui.BoldText(ui.ByteCountSI(m.payloadSize))
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
		finishedText := fmt.Sprintf("Received %d %s (%s compressed)\n\n%s", len(m.receivedFiles), oneOrMoreFiles, ui.ByteCountSI(m.payloadSize), indentedWrappedFiles)
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

// -------------------- UI COMMANDS ---------------------------

func connectCmd(addr string) tea.Cmd {
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
		temp, err := os.CreateTemp(os.TempDir(), file.RECEIVE_TEMP_FILE_NAME_PREFIX)
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
		case transfer.Type:
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
		_, err := temp.Seek(0, 0)
		if err != nil {
			return ui.ErrorMsg(err)
		}

		filenames, decompressedSize, err := file.DecompressAndUnarchiveBytes(temp)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return decompressionDoneMsg{filenames: filenames, decompressedPayloadSize: decompressedSize}
	}
}

// -------------------- HELPER METHODS -------------------------

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

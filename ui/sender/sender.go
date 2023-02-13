package sender

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/file"
	"github.com/SpatiumPortae/portal/internal/semver"
	"github.com/SpatiumPortae/portal/internal/sender"
	"github.com/SpatiumPortae/portal/protocol/transfer"
	"github.com/SpatiumPortae/portal/ui"
	"github.com/SpatiumPortae/portal/ui/transferprogress"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
)

// ------------------------------------------------------ Ui State -----------------------------------------------------

type uiState int

// flows from the top down.
const (
	showPassword uiState = iota
	showSendingProgress
	showFinished
	showError
)

// ------------------------------------------------------ Messages -----------------------------------------------------

type connectMsg struct {
	password string
	conn     conn.Rendezvous
}

type fileReadMsg struct {
	files []*os.File
	size  int64
}

type compressedMsg struct {
	payload io.Reader
	size    int64
}

type transferDoneMsg struct{}

// ------------------------------------------------------- Model -------------------------------------------------------

type Option func(m *model)

func WithVersion(version semver.Version) Option {
	return func(m *model) {
		m.version = &version
	}
}

type model struct {
	state        uiState       // defaults to 0 (showPassword)
	transferType transfer.Type // defaults to 0 (Unknown)
	errorMessage string
	readyToSend  bool

	msgs chan interface{}

	rendezvousAddr string

	password         string
	fileNames        []string
	uncompressedSize int64
	payload          io.Reader
	payloadSize      int64
	version          *semver.Version

	width            int
	spinner          spinner.Model
	transferProgress transferprogress.Model
	help             help.Model
	keys             ui.KeyMap
	copyMessageTimer timer.Model
}

// New creates a new sender program.
func New(filenames []string, addr string, opts ...Option) *tea.Program {
	m := model{
		transferProgress: transferprogress.New(),
		fileNames:        filenames,
		rendezvousAddr:   addr,
		msgs:             make(chan interface{}, 10),
		help:             help.New(),
		keys:             ui.Keys,
		copyMessageTimer: timer.NewWithInterval(ui.TEMP_UI_MESSAGE_DURATION, 100*time.Millisecond),
	}
	for _, opt := range opts {
		opt(&m)
	}
	m.resetSpinner()
	return tea.NewProgram(m)
}

func (m model) Init() tea.Cmd {
	var versionCmd tea.Cmd
	if m.version != nil {
		versionCmd = ui.VersionCmd(*m.version)
	}
	return tea.Sequence(versionCmd, tea.Batch(m.spinner.Tick, readFilesCmd(m.fileNames), connectCmd(m.rendezvousAddr)))
}

// ------------------------------------------------------- Update ------------------------------------------------------

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case ui.VersionMsg:
		var message string
		switch m.version.Compare(msg.Latest) {
		case semver.CompareNewMajor,
			semver.CompareNewMinor,
			semver.CompareNewPatch:
			//lint:ignore ST1005 error string displayed in UI
			return m, ui.ErrorCmd(fmt.Errorf("Your version is (%s) is incompatible with the latest version (%s)", m.version, msg.Latest))
		case semver.CompareOldMajor:
			message = ui.WarningText(fmt.Sprintf("New major version available (%s -> %s)", m.version, msg.Latest))
		case semver.CompareOldMinor:
			message = ui.WarningText(fmt.Sprintf("New minor version available (%s -> %s)", m.version, msg.Latest))
		case semver.CompareOldPatch:
			message = ui.WarningText(fmt.Sprintf("New patch available (%s -> %s)", m.version, msg.Latest))
		case semver.CompareEqual:
			message = ui.CheckText(fmt.Sprintf("On latest version (%s)", m.version))
		}
		return m, ui.TaskCmd(message, nil)

	case fileReadMsg:
		m.uncompressedSize = msg.size
		message := fmt.Sprintf("Read %d objects (%s)", len(m.fileNames), ui.ByteCountSI(msg.size))
		if len(m.fileNames) == 1 {
			message = fmt.Sprintf("Read %d object (%s)", len(m.fileNames), ui.ByteCountSI(msg.size))
		}
		return m, ui.TaskCmd(message, compressFilesCmd(msg.files))

	case compressedMsg:
		m.payload = msg.payload
		m.payloadSize = msg.size
		m.transferProgress.PayloadSize = msg.size
		m.readyToSend = true
		m.resetSpinner()
		message := fmt.Sprintf("Compressed objects (%s)", ui.ByteCountSI(msg.size))
		if len(m.fileNames) == 1 {
			message = fmt.Sprintf("Compressed object (%s)", ui.ByteCountSI(msg.size))
		}
		return m, ui.TaskCmd(message, m.spinner.Tick)

	case connectMsg:
		m.keys.CopyPassword.SetEnabled(true)
		m.password = msg.password
		connectMessage := fmt.Sprintf("Connected to Portal server (%s)", m.rendezvousAddr)
		return m, ui.TaskCmd(connectMessage, secureCmd(msg.conn, msg.password))

	case timer.TickMsg:
		var cmd tea.Cmd
		m.copyMessageTimer, cmd = m.copyMessageTimer.Update(msg)
		if m.copyMessageTimer.Running() {
			m.keys.CopyPassword.SetHelp(m.keys.CopyPassword.Help().Key, ui.CopyKeyActiveHelpText)
		}
		return m, cmd

	case timer.TimeoutMsg:
		var cmd tea.Cmd
		m.state = showPassword
		m.copyMessageTimer, cmd = m.copyMessageTimer.Update(msg)
		m.keys.CopyPassword.SetHelp(m.keys.CopyPassword.Help().Key, ui.CopyKeyHelpText)
		return m, cmd

	case ui.TransferTypeMsg:
		m.transferType = msg.Type
		var message string
		switch m.transferType {
		case transfer.Direct:
			message = "Using direct connection to receiver"
		case transfer.Relay:
			message = "Using relayed connection to receiver"
		}
		return m, ui.TaskCmd(message, listenTransferCmd(m.msgs))

	case ui.SecureMsg:
		// In the case we are not ready to send yet we pass on the same message.
		if !m.readyToSend {
			return m, func() tea.Msg {
				return msg
			}
		}
		cmd := tea.Batch(
			listenTransferCmd(m.msgs),
			transferCmd(msg.Conn, m.payload, m.payloadSize, m.msgs))
		return m, cmd

	case ui.TransferStateMessage:
		var message string
		switch msg.State {
		case transfer.ReceiverRequestPayload:
			m.keys.CopyPassword.SetEnabled(false)
			message = "Established encrypted connection to receiver"
		}
		return m, ui.TaskCmd(message, listenTransferCmd(m.msgs))

	case ui.ProgressMsg:
		cmds := []tea.Cmd{listenTransferCmd(m.msgs)}
		if m.state != showSendingProgress {
			m.state = showSendingProgress
			m.resetSpinner()
			m.transferProgress.StartTransfer()
			cmds = append(cmds, m.spinner.Tick)
		}
		transferProgressModel, transferProgressCmd := m.transferProgress.Update(msg)
		m.transferProgress = transferProgressModel.(transferprogress.Model)
		cmds = append(cmds, transferProgressCmd)
		return m, tea.Batch(cmds...)

	case transferDoneMsg:
		m.state = showFinished
		message := fmt.Sprintf("Transfer completed in %s with average transfer speed %s/s",
			time.Since(m.transferProgress.TransferStartTime).Round(time.Millisecond).String(),
			ui.ByteCountSI(m.transferProgress.TransferSpeedEstimateBps),
		)
		return m, ui.TaskCmd(message, ui.QuitCmd())

	case ui.ErrorMsg:
		m.state = showError
		m.errorMessage = msg.Error()
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.CopyPassword):
			err := clipboard.WriteAll(fmt.Sprintf("portal receive %s", m.password))
			if err != nil {
				return m, ui.ErrorCmd(errors.New("Failed to copy password to clipboard"))
			} else {
				m.copyMessageTimer.Timeout = ui.TEMP_UI_MESSAGE_DURATION
				cmd := m.copyMessageTimer.Init()
				return m, cmd
			}
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		transferProgressModel, cmd := m.transferProgress.Update(msg)
		m.transferProgress = transferProgressModel.(transferprogress.Model)
		return m, cmd

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

// -------------------------------------------------------- View -------------------------------------------------------

func (m model) View() string {
	// Setup strings to use in view.
	uncompressed := ui.BoldText(ui.ByteCountSI(m.uncompressedSize))
	readiness := fmt.Sprintf("%s Compressing objects (%s), preparing to send", m.spinner.View(), uncompressed)
	if m.readyToSend {
		readiness = fmt.Sprintf("%s Awaiting receiver, ready to send", m.spinner.View())
	}
	if m.state == showSendingProgress {
		readiness = fmt.Sprintf("%s Sending", m.spinner.View())
	}

	slices.Sort(m.fileNames)
	filesToSend := ui.ItalicText(strings.Join(m.fileNames, ", "))

	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%s %d object", readiness, len(m.fileNames)))
	if len(m.fileNames) > 1 {
		builder.WriteRune('s')
	}
	if m.payloadSize != 0 {
		compressed := ui.BoldText(ui.ByteCountSI(m.payloadSize))
		builder.WriteString(fmt.Sprintf(" (%s)", compressed))
	}

	switch m.transferType {
	case transfer.Direct:
		builder.WriteString(" using direct transfer")
	case transfer.Relay:
		builder.WriteString(" using relayed transfer")
	case transfer.Unknown:
	}

	indentedWrappedFiles := indent.String(wordwrap.String(fmt.Sprintf("Sending: %s", filesToSend), ui.MAX_WIDTH), ui.PADDING)
	builder.WriteString("\n\n")
	builder.WriteString(indentedWrappedFiles)
	fileInfoText := builder.String()

	switch m.state {
	case showPassword:
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(fileInfoText) + "\n\n" +
			ui.PadText + ui.InfoStyle("On the receiving end, run:") + "\n" +
			ui.PadText + ui.InfoStyle(fmt.Sprintf("portal receive %s", m.password)) + "\n\n" +
			ui.PadText + m.help.View(m.keys) + "\n\n"

	case showSendingProgress:
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(fileInfoText) + "\n\n" +
			ui.PadText + m.transferProgress.View() + "\n\n" +
			ui.PadText + m.help.View(m.keys) + "\n\n"

	case showFinished:
		finishedText := fmt.Sprintf("Sent %d object(s) (%s compressed)", len(m.fileNames), ui.ByteCountSI(m.payloadSize))
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(finishedText) + "\n\n" +
			ui.PadText + m.transferProgress.View() + "\n\n"

	case showError:
		return ui.ErrorText(m.errorMessage)

	default:
		return ""
	}
}

// ------------------------------------------------------ Commands -----------------------------------------------------

// connectCmd command that connects to the rendezvous server.
func connectCmd(addr string) tea.Cmd {
	return func() tea.Msg {
		rc, password, err := sender.ConnectRendezvous(addr)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return connectMsg{password: password, conn: rc}
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
		return transferDoneMsg{}
	}
}

// readFilesCmd command that reads the files from the provided paths.
func readFilesCmd(paths []string) tea.Cmd {
	return func() tea.Msg {
		files, err := file.ReadFiles(paths)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		size, err := file.FilesTotalSize(files)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return fileReadMsg{files: files, size: size}
	}
}

// compressFilesCmd is a command that compresses and archives the
// provided files.
func compressFilesCmd(files []*os.File) tea.Cmd {
	return func() tea.Msg {
		tar, size, err := file.ArchiveAndCompressFiles(files)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return compressedMsg{payload: tar, size: size}
	}
}

// listenTransferCmd is a command that listens to the provided
// channel and formats messages.
func listenTransferCmd(msgs chan interface{}) tea.Cmd {
	return func() tea.Msg {
		msg := <-msgs
		switch v := msg.(type) {
		case transfer.Type:
			return ui.TransferTypeMsg{Type: v}
		case transfer.MsgType:
			return ui.TransferStateMessage{State: v}
		case int:
			return ui.ProgressMsg(v)
		default:
			return nil
		}
	}
}

// -------------------- HELPER METHODS -------------------------

func (m *model) resetSpinner() {
	m.spinner = spinner.New()
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

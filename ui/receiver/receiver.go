package receiver

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/file"
	"github.com/SpatiumPortae/portal/internal/receiver"
	"github.com/SpatiumPortae/portal/internal/semver"
	"github.com/SpatiumPortae/portal/protocol/transfer"
	"github.com/SpatiumPortae/portal/ui"
	"github.com/SpatiumPortae/portal/ui/transferprogress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	"golang.org/x/exp/slices"
)

// ------------------------------------------------------ Ui State -----------------------------------------------------
type uiState int

// Flows from the top down.
const (
	showEstablishing uiState = iota
	showReceivingProgress
	showDecompressing
	showFinished
	showError
)

// ------------------------------------------------------ Messages -----------------------------------------------------
type connectMsg struct {
	conn conn.Rendezvous
}

type payloadSizeMsg struct {
	size int64
}

type transferHandshakeMsg struct {
	receiver receiver.Receiver
}

type receiveDoneMsg struct {
	temp *os.File
}

type decompressionDoneMsg struct {
	filenames               []string
	decompressedPayloadSize int64
}

// ------------------------------------------------------- Model -------------------------------------------------------

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
	errorMessage string

	rendezvousAddr string

	receivedFiles           []string
	payloadSize             int64
	decompressedPayloadSize int64
	transferStartTime       time.Time
	version                 *semver.Version

	width       int
	spinner     spinner.Model
	progressBar transferprogress.Model
}

// New creates a receiver program.
func New(addr string, password string, opts ...Option) *tea.Program {
	m := model{
		progressBar:    transferprogress.New(),
		password:       password,
		rendezvousAddr: addr,
	}
	for _, opt := range opts {
		opt(&m)
	}
	m.resetSpinner()
	p := tea.NewProgram(m)
	transferprogress.Init(p)
	return p
}

func (m model) Init() tea.Cmd {
	if m.version == nil {
		return tea.Batch(spinner.Tick, m.connectCmd())
	}
	return ui.VersionCmd(*m.version)
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
			return m, ui.ErrorCmd(fmt.Errorf("Your version is (%s) is incompatible with the latest version (%s)", m.version, msg.Latest))
		case semver.CompareOldMajor:
			return m, ui.ErrorCmd(fmt.Errorf("New major version available (%s -> %s)", m.version, msg.Latest))
		case semver.CompareOldMinor:
			message = ui.WarningText(fmt.Sprintf("New minor version available (%s -> %s)", m.version, msg.Latest))
		case semver.CompareOldPatch:
			message = ui.WarningText(fmt.Sprintf("New patch available (%s -> %s)", m.version, msg.Latest))
		case semver.CompareEqual:
			message = ui.CheckText(fmt.Sprintf("You have the latest version (%s)", m.version))
		default:
		}
		return m, ui.TaskCmd(message, spinner.Tick, m.connectCmd())

	case connectMsg:
		message := fmt.Sprintf("Connected to Portal server (%s)", m.rendezvousAddr)
		return m, ui.TaskCmd(message, m.secureCmd(msg.conn), spinner.Tick)

	case ui.SecureMsg:
		message := "Established encrypted connection with receiver"
		return m, ui.TaskCmd(message, m.transferHandshakeCmd(msg.Conn))

	case transferHandshakeMsg:
		m.transferType = msg.receiver.Type()
		var message string
		switch m.transferType {
		case transfer.Direct:
			message = "Using direct connection to sender"
		case transfer.Relay:
			message = "Using relayed connection to sender"
		}
		m.progressBar.PayloadSize = msg.receiver.PayloadSize()
		m.payloadSize = msg.receiver.PayloadSize()
		m.state = showReceivingProgress
		m.resetSpinner()
		return m, ui.TaskCmd(message, m.receiveCmd(msg.receiver), m.spinner.Tick)

	case ui.ProgressMsg:
		transferProgressModel, transferProgressCmd := m.progressBar.Update(msg)
		m.progressBar = transferProgressModel.(transferprogress.Model)
		return m, tea.Batch(transferProgressCmd, m.spinner.Tick)

	case receiveDoneMsg:
		m.state = showDecompressing
		message := fmt.Sprintf("Transfer completed in %s with average transfer speed %s/s",
			time.Since(*m.progressBar.TransferStartTime).Round(time.Millisecond).String(),
			ui.ByteCountSI(m.progressBar.TransferSpeedEstimateBps),
		)
		m.resetSpinner()
		return m, ui.TaskCmd(message, decompressCmd(msg.temp), m.spinner.Tick)

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
		m.width = msg.Width
		transferProgressModel, cmd := m.progressBar.Update(msg)
		m.progressBar = transferProgressModel.(transferprogress.Model)
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
		receivingText := fmt.Sprintf("%s Receiving objects (total size %s) using %s transfer", m.spinner.View(), payloadSize, transferType)
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(receivingText) + "\n\n" +
			ui.PadText + m.progressBar.View() + "\n\n" +
			ui.PadText + ui.QuitCommandsHelpText + "\n\n"

	case showDecompressing:
		payloadSize := ui.BoldText(ui.ByteCountSI(m.payloadSize))
		decompressingText := fmt.Sprintf("%s Decompressing payload (%s compressed) and writing to disk", m.spinner.View(), payloadSize)
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(decompressingText) + "\n\n" +
			ui.PadText + m.progressBar.View() + "\n\n" +
			ui.PadText + ui.QuitCommandsHelpText + "\n\n"

	case showFinished:
		indentedWrappedFiles := indent.String(fmt.Sprintf("Received: %s", wordwrap.String(ui.ItalicText(ui.TopLevelFilesText(m.receivedFiles)), ui.MAX_WIDTH)), ui.PADDING)

		var oneOrMoreFiles string
		if len(m.receivedFiles) > 1 {
			oneOrMoreFiles = "objects"
		} else {
			oneOrMoreFiles = "object"
		}
		finishedText := fmt.Sprintf("Received %d %s (%s compressed)\n\n%s", len(m.receivedFiles), oneOrMoreFiles, ui.ByteCountSI(m.payloadSize), indentedWrappedFiles)
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(finishedText) + "\n\n" +
			ui.PadText + m.progressBar.View() + "\n\n"

	case showError:
		return ui.ErrorText(m.errorMessage)

	default:
		return ""
	}
}

// -------------------- UI COMMANDS ---------------------------

func (m *model) connectCmd() tea.Cmd {
	return func() tea.Msg {
		rc, err := receiver.ConnectRendezvous(m.rendezvousAddr)
		if err != nil {
			return ui.ErrorMsg(fmt.Errorf("connection to relay: %w", err))
		}
		return connectMsg{conn: rc}
	}
}

func (m *model) secureCmd(rc conn.Rendezvous) tea.Cmd {
	return func() tea.Msg {
		tc, err := receiver.SecureConnection(rc, m.password)
		if err != nil {
			return ui.ErrorMsg(fmt.Errorf("establish secure connection: %w", err))
		}
		return ui.SecureMsg{Conn: tc}
	}
}

func (m *model) transferHandshakeCmd(tc conn.Transfer) tea.Cmd {
	return func() tea.Msg {
		receiver, err := receiver.TransferHandshake(tc, transferprogress.Writer)
		if err != nil {
			return ui.ErrorMsg(fmt.Errorf("executing transfer handshake: %w", err))
		}
		return transferHandshakeMsg{
			receiver: receiver,
		}
	}
}

func (m *model) receiveCmd(receiver receiver.Receiver) tea.Cmd {
	return func() tea.Msg {
		temp, err := os.CreateTemp(os.TempDir(), file.RECEIVE_TEMP_FILE_NAME_PREFIX)
		if err != nil {
			return ui.ErrorMsg(fmt.Errorf("creating temporary file: %w", err))
		}
		if err := receiver.Receive(temp); err != nil {
			return ui.ErrorMsg(fmt.Errorf("receiving payload: %w", err))
		}
		return receiveDoneMsg{temp: temp}
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
			return ui.ErrorMsg(fmt.Errorf("decompressing files:%w", err))
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

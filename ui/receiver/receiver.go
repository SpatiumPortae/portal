package receiver

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/file"
	"github.com/SpatiumPortae/portal/internal/receiver"
	"github.com/SpatiumPortae/portal/internal/semver"
	"github.com/SpatiumPortae/portal/protocol/transfer"
	"github.com/SpatiumPortae/portal/ui"
	"github.com/SpatiumPortae/portal/ui/filetable"
	"github.com/SpatiumPortae/portal/ui/transferprogress"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ------------------------------------------------------ Ui State -----------------------------------------------------
type uiState int

// Flows from the top down.
const (
	showEstablishing uiState = iota
	showReceivingProgress
	showDecompressing
	showFinished
)

// ------------------------------------------------------ Messages -----------------------------------------------------
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

	ctx  context.Context
	msgs chan interface{}

	rendezvousAddr string

	receivedFiles           []string
	payloadSize             int64
	decompressedPayloadSize int64
	version                 *semver.Version

	width            int
	spinner          spinner.Model
	transferProgress transferprogress.Model
	fileTable        filetable.Model
	help             help.Model
	keys             ui.KeyMap
}

// New creates a new receiver program.
func New(addr string, password string, opts ...Option) *tea.Program {
	m := model{
		transferProgress: transferprogress.New(),
		msgs:             make(chan interface{}, 10),
		fileTable:        filetable.New(),
		password:         password,
		rendezvousAddr:   addr,
		help:             help.New(),
		keys:             ui.Keys,
		ctx:              context.Background(),
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
		versionCmd = ui.VersionCmd(m.ctx, m.rendezvousAddr)
	}
	return tea.Sequence(versionCmd, tea.Batch(m.spinner.Tick, connectCmd(m.rendezvousAddr)))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.VersionMsg:
		var message string
		switch m.version.Compare(msg.ServerVersion) {
		case semver.CompareNewMajor,
			semver.CompareOldMajor:
			//lint:ignore ST1005 error string displayed in UI
			return m, ui.ErrorCmd(fmt.Errorf("Portal version (%s) incompatible with server version (%s)", m.version, msg.ServerVersion))
		case semver.CompareNewMinor,
			semver.CompareNewPatch:
			message = ui.WarningText(fmt.Sprintf("Portal version (%s) newer than server version (%s)", m.version, msg.ServerVersion))
		case semver.CompareOldMinor,
			semver.CompareOldPatch:
			message = ui.WarningText(fmt.Sprintf("Server version (%s) newer than Portal version (%s)", m.version, msg.ServerVersion))
		case semver.CompareEqual:
			message = ui.SuccessText(fmt.Sprintf("Portal version (%s) compatible with server version (%s)", m.version, msg.ServerVersion))
		}
		return m, ui.TaskCmd(message, nil)

	case connectMsg:
		message := fmt.Sprintf("Connected to Portal server (%s)", m.rendezvousAddr)
		return m, ui.TaskCmd(message, secureCmd(m.ctx, msg.conn, m.password))

	case ui.SecureMsg:
		message := "Established encrypted connection to sender"
		return m, ui.TaskCmd(message,
			tea.Batch(listenReceiveCmd(m.msgs), receiveCmd(m.ctx, msg.Conn, m.msgs)))

	case payloadSizeMsg:
		m.payloadSize = msg.size
		m.transferProgress.PayloadSize = msg.size
		return m, listenReceiveCmd(m.msgs)

	case ui.TransferTypeMsg:
		var message string
		m.transferType = msg.Type
		switch m.transferType {
		case transfer.Direct:
			message = "Using direct connection to sender"
		case transfer.Relay:
			message = "Using relayed connection to sender"
		}
		return m, ui.TaskCmd(message, listenReceiveCmd(m.msgs))

	case ui.ProgressMsg:
		cmds := []tea.Cmd{listenReceiveCmd(m.msgs)}
		if m.state != showReceivingProgress {
			m.state = showReceivingProgress
			m.resetSpinner()
			m.transferProgress.StartTransfer()
			cmds = append(cmds, m.spinner.Tick)
		}
		transferProgressModel, transferProgressCmd := m.transferProgress.Update(msg)
		m.transferProgress = transferProgressModel.(transferprogress.Model)
		cmds = append(cmds, transferProgressCmd)
		return m, tea.Batch(cmds...)

	case receiveDoneMsg:
		m.state = showDecompressing
		m.resetSpinner()
		message := fmt.Sprintf("Transfer completed in %s with average transfer speed %s/s",
			time.Since(m.transferProgress.TransferStartTime).Round(time.Millisecond).String(),
			ui.ByteCountSI(m.transferProgress.TransferSpeedEstimateBps),
		)

		m.fileTable.SetMaxHeight(math.MaxInt)
		m.fileTable = m.fileTable.Finalize().(filetable.Model)
		return m, ui.TaskCmd(message, tea.Batch(m.spinner.Tick, decompressCmd(msg.temp)))

	case decompressionDoneMsg:
		m.state = showFinished
		m.receivedFiles = msg.filenames
		m.decompressedPayloadSize = msg.decompressedPayloadSize

		m.fileTable.SetFiles(m.receivedFiles)
		return m, ui.QuitCmd()

	case ui.ErrorMsg:
		return m, ui.ErrorCmd(errors.New(msg.Error()))

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}

		fileTableModel, fileTableCmd := m.fileTable.Update(msg)
		m.fileTable = fileTableModel.(filetable.Model)

		return m, fileTableCmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		transferProgressModel, transferProgressCmd := m.transferProgress.Update(msg)
		m.transferProgress = transferProgressModel.(transferprogress.Model)
		fileTableModel, fileTableCmd := m.fileTable.Update(msg)
		m.fileTable = fileTableModel.(filetable.Model)
		return m, tea.Batch(transferProgressCmd, fileTableCmd)

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m model) View() string {

	switch m.state {

	case showEstablishing:
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(fmt.Sprintf("%s Establishing connection with sender", m.spinner.View())) + "\n\n" +
			ui.PadText + m.help.View(m.keys) + "\n\n"

	case showReceivingProgress:
		var transferType string
		if m.transferType == transfer.Direct {
			transferType = "direct"
		} else {
			transferType = "relayed"
		}

		payloadSize := ui.BoldText(ui.ByteCountSI(m.payloadSize))
		receivingText := fmt.Sprintf("%s Receiving objects (%s) using %s transfer", m.spinner.View(), payloadSize, transferType)
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(receivingText) + "\n\n" +
			ui.PadText + m.transferProgress.View() + "\n\n" +
			ui.PadText + m.help.View(m.keys) + "\n\n"

	case showDecompressing:
		payloadSize := ui.BoldText(ui.ByteCountSI(m.payloadSize))
		decompressingText := fmt.Sprintf("%s Decompressing payload (%s compressed) and writing to disk", m.spinner.View(), payloadSize)
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(decompressingText) + "\n\n" +
			ui.PadText + m.transferProgress.View() + "\n\n" +
			ui.PadText + m.help.View(m.keys) + "\n\n"

	case showFinished:
		oneOrMoreFiles := "object"
		if len(m.receivedFiles) > 1 {
			oneOrMoreFiles += "s"
		}
		finishedText := fmt.Sprintf("Received %d %s (%s compressed)", len(m.receivedFiles), oneOrMoreFiles, ui.ByteCountSI(m.payloadSize))
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(finishedText) + "\n\n" +
			ui.PadText + m.transferProgress.View() + "\n\n" +
			m.fileTable.View()

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

func secureCmd(ctx context.Context, rc conn.Rendezvous, password string) tea.Cmd {
	return func() tea.Msg {
		tc, err := receiver.SecureConnection(ctx, rc, password)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return ui.SecureMsg{Conn: tc}
	}
}

func receiveCmd(ctx context.Context, tc conn.Transfer, msgs ...chan interface{}) tea.Cmd {
	return func() tea.Msg {
		temp, err := os.CreateTemp(os.TempDir(), file.RECEIVE_TEMP_FILE_NAME_PREFIX)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		if err := receiver.Receive(ctx, tc, temp, msgs...); err != nil {
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
		case transfer.MsgType:
			return ui.TransferStateMessage{State: v}
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
	m.spinner = spinner.New()
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ELEMENT_COLOR))
	if m.state == showEstablishing {
		m.spinner.Spinner = ui.WaitingSpinner
	}
	if m.state == showDecompressing {
		m.spinner.Spinner = ui.CompressingSpinner
	}
	if m.state == showReceivingProgress {
		m.spinner.Spinner = ui.ReceivingSpinner
	}
}

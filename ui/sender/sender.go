package sender

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/SpatiumPortae/portal/internal/config"
	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/file"
	"github.com/SpatiumPortae/portal/internal/semver"
	"github.com/SpatiumPortae/portal/internal/sender"
	"github.com/SpatiumPortae/portal/protocol/transfer"
	"github.com/SpatiumPortae/portal/ui"
	"github.com/SpatiumPortae/portal/ui/filetable"
	"github.com/SpatiumPortae/portal/ui/transferprogress"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

// ------------------------------------------------------ Ui State -----------------------------------------------------

type uiState int

// flows from the top down.
const (
	showPassword uiState = iota
	showSendingProgress
	showFinished
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

func WithCopyFlags(flags map[string]string) Option {
	return func(m *model) {
		m.copyFlags = flags
	}
}

type model struct {
	state        uiState       // defaults to 0 (showPassword)
	transferType transfer.Type // defaults to 0 (Unknown)
	readyToSend  bool
	ctx          context.Context

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
	fileTable        filetable.Model
	help             help.Model
	keys             ui.KeyMap
	copyMessageTimer timer.Model
	copyFlags        map[string]string
}

// New creates a new sender program.
func New(filenames []string, addr string, opts ...Option) *tea.Program {
	m := model{
		transferProgress: transferprogress.New(),
		fileTable:        filetable.New(filetable.WithFiles(filenames)),
		fileNames:        filenames,
		rendezvousAddr:   addr,
		msgs:             make(chan interface{}, 10),
		help:             help.New(),
		keys:             ui.Keys,
		copyMessageTimer: timer.NewWithInterval(ui.TEMP_UI_MESSAGE_DURATION, 100*time.Millisecond),
		ctx:              context.Background(),
	}
	m.keys.FileListUp.SetEnabled(true)
	m.keys.FileListDown.SetEnabled(true)
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
	return tea.Sequence(versionCmd, tea.Batch(m.spinner.Tick, readFilesCmd(m.fileNames), connectCmd(m.ctx, m.rendezvousAddr)))
}

// ------------------------------------------------------- Update ------------------------------------------------------

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
			message = ui.WarningText(fmt.Sprintf("Server version (%s) newer than Portal version (%s)", msg.ServerVersion, m.version))
		case semver.CompareEqual:
			message = ui.SuccessText(fmt.Sprintf("Portal version (%s) compatible with server version (%s)", m.version, msg.ServerVersion))
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
		return m, ui.TaskCmd(connectMessage, secureCmd(m.ctx, msg.conn, msg.password))

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
			transferCmd(m.ctx, msg.Conn, m.payload, m.payloadSize, m.msgs))
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

		m.fileTable = m.fileTable.Finalize().(filetable.Model)
		return m, ui.TaskCmd(message, ui.QuitCmd())

	case ui.ErrorMsg:
		return m, ui.ErrorCmd(errors.New(msg.Error()))

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.CopyPassword):
			err := clipboard.WriteAll(m.copyReceiverCommand())
			if err != nil {
				return m, ui.ErrorCmd(errors.New("Failed to copy password to clipboard"))
			} else {
				m.copyMessageTimer.Timeout = ui.TEMP_UI_MESSAGE_DURATION
				cmd := m.copyMessageTimer.Init()
				return m, cmd
			}
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

	statusText := builder.String()

	switch m.state {
	case showPassword:
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(statusText) + "\n\n" +
			ui.PadText + ui.InfoStyle("On the receiving end, run:") + "\n" +
			ui.PadText + ui.InfoStyle(m.copyReceiverCommand()) + "\n\n" +
			m.fileTable.View() +
			ui.PadText + m.help.View(m.keys) + "\n\n"

	case showSendingProgress:
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(statusText) + "\n\n" +
			ui.PadText + m.transferProgress.View() + "\n\n" +
			m.fileTable.View() +
			ui.PadText + m.help.View(m.keys) + "\n\n"

	case showFinished:
		finishedText := fmt.Sprintf("Sent %d object(s) (%s compressed)", len(m.fileNames), ui.ByteCountSI(m.payloadSize))
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(finishedText) + "\n\n" +
			ui.PadText + m.transferProgress.View() + "\n\n" +
			m.fileTable.View()

	default:
		return ""
	}
}

// ------------------------------------------------------ Commands -----------------------------------------------------

// connectCmd command that connects to the rendezvous server.
func connectCmd(ctx context.Context, addr string) tea.Cmd {
	return func() tea.Msg {
		rc, password, err := sender.ConnectRendezvous(ctx, addr)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return connectMsg{password: password, conn: rc}
	}
}

// secureCmd command that secures a connection for transfer.
func secureCmd(ctx context.Context, rc conn.Rendezvous, password string) tea.Cmd {
	return func() tea.Msg {
		tc, err := sender.SecureConnection(ctx, rc, password)
		if err != nil {
			return ui.ErrorMsg(err)
		}
		return ui.SecureMsg{Conn: tc}
	}
}

// transferCmd command that does the transfer sequence.
// The msgs channel is used to provide intermediate messages to the ui.
func transferCmd(ctx context.Context, tc conn.Transfer, payload io.Reader, payloadSize int64, msgs ...chan interface{}) tea.Cmd {
	return func() tea.Msg {
		err := sender.Transfer(ctx, tc, payload, payloadSize, msgs...)
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

		var totalSize int64
		for _, f := range files {
			size, err := file.FileSize(f.Name())
			if err != nil {
				return ui.ErrorMsg(err)
			}
			totalSize += size
		}

		return fileReadMsg{files: files, size: totalSize}
	}
}

// compressFilesCmd is a command that compresses and archives the
// provided files.
func compressFilesCmd(files []*os.File) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			for _, f := range files {
				f.Close()
			}
		}()
		tar, size, err := file.PackFiles(files)
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

// -------------------------------------------------- Helper Functions -------------------------------------------------

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

func (m *model) copyReceiverCommand() string {
	var builder strings.Builder
	builder.WriteString("portal receive ")
	builder.WriteString(m.password)

	relayAddrKey := "relay"
	if !config.IsDefault(relayAddrKey) {
		builder.WriteRune(' ')
		builder.WriteString(fmt.Sprintf("--%s", relayAddrKey))
		builder.WriteRune(' ')
		builder.WriteString(viper.GetString(relayAddrKey))
	}

	return builder.String()
}

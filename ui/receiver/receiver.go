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
	"github.com/erikgeiser/promptkit"
	"github.com/erikgeiser/promptkit/confirmation"
	"github.com/spf13/viper"
)

// ------------------------------------------------------ Ui State -----------------------------------------------------
type uiState int

// Flows from the top down.
const (
	showEstablishing uiState = iota
	showReceivingProgress
	showDecompressing
	showOverwritePrompt
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

type overwritePromptRequestMsg struct {
	fileName string
}

type overwritePromptResponseMsg struct {
	shouldOverwrite bool
}

type decompressionDoneMsg struct {
	fileNames               []string
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

	ctx                      context.Context
	msgs                     chan interface{}
	overwritePromptRequests  chan overwritePromptRequestMsg
	overwritePromptResponses chan overwritePromptResponseMsg

	rendezvousAddr string

	receivedFiles           []string
	payloadSize             int64
	decompressedPayloadSize int64
	version                 *semver.Version

	width            int
	spinner          spinner.Model
	transferProgress transferprogress.Model
	fileTable        filetable.Model
	overwritePrompt  confirmation.Model
	help             help.Model
	keys             ui.KeyMap
}

// New creates a new receiver program.
func New(addr string, password string, opts ...Option) *tea.Program {
	m := model{
		transferProgress:         transferprogress.New(),
		msgs:                     make(chan interface{}, 10),
		overwritePromptRequests:  make(chan overwritePromptRequestMsg),
		overwritePromptResponses: make(chan overwritePromptResponseMsg),
		password:                 password,
		rendezvousAddr:           addr,
		fileTable:                filetable.New(),
		overwritePrompt:          *confirmation.NewModel(confirmation.New("", confirmation.Undecided)),
		help:                     help.New(),
		keys:                     ui.Keys,
		ctx:                      context.Background(),
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
			message = ui.WarningText(fmt.Sprintf("Server version (%s) newer than Portal version (%s)", msg.ServerVersion, m.version))
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

		cmds := []tea.Cmd{m.spinner.Tick,
			m.listenOverwritePromptRequestsCmd(),
			m.decompressCmd(msg.temp),
		}

		return m, ui.TaskCmd(message, tea.Batch(cmds...))

	case overwritePromptRequestMsg:
		m.state = showOverwritePrompt
		m.resetSpinner()
		m.keys.OverwritePromptYes.SetEnabled(true)
		m.keys.OverwritePromptNo.SetEnabled(true)
		m.keys.OverwritePromptConfirm.SetEnabled(true)

		return m, tea.Batch(m.spinner.Tick, m.newOverwritePrompt(msg.fileName))

	case decompressionDoneMsg:
		m.state = showFinished
		m.receivedFiles = msg.fileNames
		m.decompressedPayloadSize = msg.decompressedPayloadSize

		m.fileTable.SetFiles(m.receivedFiles)
		return m, ui.QuitCmd()

	case ui.ErrorMsg:
		return m, ui.ErrorCmd(errors.New(msg.Error()))

	case tea.KeyMsg:
		var cmds []tea.Cmd
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}

		fileTableModel, fileTableCmd := m.fileTable.Update(msg)
		m.fileTable = fileTableModel.(filetable.Model)
		cmds = append(cmds, fileTableCmd)

		_, promptCmd := m.overwritePrompt.Update(msg)
		if m.state == showOverwritePrompt {
			switch msg.String() {
			case "left", "right":
				cmds = append(cmds, promptCmd)
			}
			switch {
			case key.Matches(msg, m.keys.OverwritePromptYes, m.keys.OverwritePromptNo, m.keys.OverwritePromptConfirm):
				m.state = showDecompressing
				m.keys.OverwritePromptYes.SetEnabled(false)
				m.keys.OverwritePromptNo.SetEnabled(false)
				m.keys.OverwritePromptConfirm.SetEnabled(false)
				shouldOverwrite, _ := m.overwritePrompt.Value()
				m.overwritePromptResponses <- overwritePromptResponseMsg{shouldOverwrite}
				cmds = append(cmds, m.listenOverwritePromptRequestsCmd())
			}
		}

		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		transferProgressModel, transferProgressCmd := m.transferProgress.Update(msg)
		m.transferProgress = transferProgressModel.(transferprogress.Model)

		fileTableModel, fileTableCmd := m.fileTable.Update(msg)
		m.fileTable = fileTableModel.(filetable.Model)

		m.overwritePrompt.MaxWidth = msg.Width - 2*ui.MARGIN - 4
		_, promptCmd := m.overwritePrompt.Update(msg)

		return m, tea.Batch(transferProgressCmd, fileTableCmd, promptCmd)

	default:
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		_, promptCmd := m.overwritePrompt.Update(msg)
		return m, tea.Batch(spinnerCmd, promptCmd)
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

	case showOverwritePrompt:
		waitingText := fmt.Sprintf("%s Waiting for file overwrite confirmation", m.spinner.View())
		return ui.PadText + ui.LogSeparator(m.width) +
			ui.PadText + ui.InfoStyle(waitingText) + "\n\n" +
			ui.PadText + m.transferProgress.View() + "\n\n" +
			ui.PadText + m.overwritePrompt.View() + "\n\n" +
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
		if len(m.receivedFiles) == 0 || len(m.receivedFiles) > 1 {
			oneOrMoreFiles += "s"
		}
		finishedText := fmt.Sprintf("Received %d %s (%s decompressed)", len(m.receivedFiles), oneOrMoreFiles, ui.ByteCountSI(m.decompressedPayloadSize))
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

func (m *model) listenOverwritePromptRequestsCmd() tea.Cmd {
	return func() tea.Msg {
		return <-m.overwritePromptRequests
	}
}

func (m *model) decompressCmd(temp *os.File) tea.Cmd {
	return func() tea.Msg {
		// Reset file position for reading.
		_, err := temp.Seek(0, 0)
		if err != nil {
			return ui.ErrorMsg(err)
		}

		// promptFunc is a no-op if we allow overwriting files without prompts.
		promptFunc := func(fileName string) (bool, error) { return true, nil }
		if viper.GetBool("prompt_overwrite_files") {
			promptFunc = func(fileName string) (bool, error) {
				m.overwritePromptRequests <- overwritePromptRequestMsg{fileName}
				overwritePromptResponse := <-m.overwritePromptResponses
				return overwritePromptResponse.shouldOverwrite, nil
			}
		}

		fileNames, size, err := file.UnpackFiles(temp, promptFunc)
		if err != nil {
			return ui.ErrorMsg(err)
		}

		return decompressionDoneMsg{fileNames, size}
	}
}

// -------------------- HELPER METHODS -------------------------

func (m *model) newOverwritePrompt(fileName string) tea.Cmd {
	prompt := confirmation.New(fmt.Sprintf("Overwrite file '%s'?", fileName), confirmation.Yes)
	m.overwritePrompt = *confirmation.NewModel(prompt)
	m.overwritePrompt.MaxWidth = m.width
	m.overwritePrompt.WrapMode = promptkit.HardWrap
	m.overwritePrompt.Template = confirmation.TemplateYN
	m.overwritePrompt.ResultTemplate = confirmation.ResultTemplateYN
	m.overwritePrompt.KeyMap.Abort = []string{}
	m.overwritePrompt.KeyMap.Toggle = []string{}
	return m.overwritePrompt.Init()
}

func (m *model) resetSpinner() {
	m.spinner = spinner.New()
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ELEMENT_COLOR))
	if m.state == showEstablishing || m.state == showOverwritePrompt {
		m.spinner.Spinner = ui.WaitingSpinner
	}
	if m.state == showDecompressing {
		m.spinner.Spinner = ui.CompressingSpinner
	}
	if m.state == showReceivingProgress {
		m.spinner.Spinner = ui.ReceivingSpinner
	}
}

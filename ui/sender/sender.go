package senderui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ZinoKader/portal/tools"
	"github.com/ZinoKader/portal/ui"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
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

type senderUIModel struct {
	state        uiState
	fileNames    []string
	payloadSize  int64
	password     string
	readyToSend  bool
	spinner      spinner.Model
	progressBar  progress.Model
	errorMessage string
}

type ReadyMsg struct{}

type PasswordMsg struct {
	Password string
}

func NewSenderUI() *tea.Program {
	m := senderUIModel{progressBar: ui.ProgressBar}
	m.resetSpinner()
	return tea.NewProgram(m)
}

func (senderUIModel) Init() tea.Cmd {
	return spinner.Tick
}

func (m senderUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case ui.FileInfoMsg:
		m.fileNames = msg.FileNames
		m.payloadSize = msg.Bytes
		return m, nil

	case ReadyMsg:
		m.readyToSend = true
		m.resetSpinner()
		return m, spinner.Tick

	case PasswordMsg:
		m.password = msg.Password
		return m, nil

	case ui.ProgressMsg:
		if m.state != showSendingProgress {
			m.state = showSendingProgress
			m.resetSpinner()
			return m, spinner.Tick
		}
		if m.progressBar.Percent() == 1.0 {
			return m, nil
		}
		cmd := m.progressBar.SetPercent(float64(msg.Progress))
		return m, cmd

	case ui.FinishedMsg:
		m.state = showFinished
		cmd := m.progressBar.SetPercent(1.0)
		return m, cmd

	case ui.ErrorMsg:
		m.state = showError
		m.errorMessage = msg.Message
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

func (m senderUIModel) View() string {

	readiness := fmt.Sprintf("%s Compressing objects, preparing to send", m.spinner.View())
	if m.readyToSend {
		readiness = fmt.Sprintf("%s Awaiting receiver, ready to send", m.spinner.View())
	}
	if m.state == showSendingProgress {
		readiness = fmt.Sprintf("%s Sending", m.spinner.View())
	}

	fileInfoText := fmt.Sprintf("%s object(s)...", readiness)
	if m.fileNames != nil && m.payloadSize != 0 {
		sort.Strings(m.fileNames)
		filesToSend := ui.ItalicText(strings.Join(m.fileNames, ", "))
		payloadSize := ui.BoldText(tools.ByteCountSI(m.payloadSize))
		fileInfoText = fmt.Sprintf("%s %d objects (%s)", readiness, len(m.fileNames), payloadSize)

		indentedWrappedFiles := indent.String(wordwrap.String(fmt.Sprintf("Sending: %s", filesToSend), ui.MAX_WIDTH), ui.PADDING)
		fileInfoText = fmt.Sprintf("%s\n\n%s", fileInfoText, indentedWrappedFiles)
	}

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
		payloadSize := ui.BoldText(tools.ByteCountSI(m.payloadSize))
		indentedWrappedFiles := indent.String(fmt.Sprintf("Sent: %s", wordwrap.String(ui.ItalicText(ui.TopLevelFilesText(m.fileNames)), ui.MAX_WIDTH)), ui.PADDING)
		finishedText := fmt.Sprintf("Sent %d objects (%s decompressed)\n\n%s", len(m.fileNames), payloadSize, indentedWrappedFiles)
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

func (m *senderUIModel) resetSpinner() {
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

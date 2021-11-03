package senderui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	s := spinner.NewModel()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.SPINNER_COLOR))
	m := senderUIModel{
		spinner:     s,
		progressBar: progress.NewModel(progress.WithDefaultGradient()),
	}
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
		return m, nil

	case PasswordMsg:
		m.password = msg.Password
		return m, nil

	case ui.ProgressMsg:
		m.state = showSendingProgress
		if m.progressBar.Percent() == 1.0 {
			return m, nil
		}
		cmd := m.progressBar.SetPercent(float64(msg.Progress))
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
		m.progressBar.Width = msg.Width - 2*ui.Padding - 4
		if m.progressBar.Width > ui.MaxWidth {
			m.progressBar.Width = ui.MaxWidth
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
	pad := strings.Repeat(" ", ui.Padding)

	readiness := fmt.Sprintf("%s Compressing files, preparing to send", m.spinner.View())
	if m.readyToSend {
		readiness = fmt.Sprintf("%s Awaiting receiver, ready to send", m.spinner.View())
	}
	if m.state == showSendingProgress {
		readiness = "Connected! Sending"
	}

	fileInfoText := fmt.Sprintf("%s file(s)...", readiness)
	if m.fileNames != nil && m.payloadSize != 0 {
		filesToSend := ui.ItalicText(strings.Join(m.fileNames, ", "))
		payloadSize := ui.BoldText(tools.ByteCountSI(m.payloadSize))
		fileInfoText = fmt.Sprintf("%s %s (%s)", readiness, filesToSend, payloadSize)
	}

	switch m.state {

	case showPassword, showPasswordWithCopy:
		copyText := "(password copied to clipboard)"
		if m.state == showPasswordWithCopy {
			copyText = "(press 'c' to copy the command to your clipboard)"
		}
		return "\n" +
			pad + ui.InfoStyle(fileInfoText) + "\n\n" +
			pad + ui.InfoStyle("On the receiving end, run:") + "\n" +
			pad + ui.InfoStyle(fmt.Sprintf("portal receive %s", m.password)) + "\n\n" +
			pad + ui.HelpStyle(copyText) + "\n\n"

	case showSendingProgress:
		quitCommandsHelp := ui.HelpStyle(fmt.Sprintf("(any of [%s] to abort)", (strings.Join(ui.QuitKeys, ", "))))
		return "\n" +
			pad + ui.InfoStyle(fileInfoText) + "\n\n" +
			pad + m.progressBar.View() + "\n\n" +
			pad + quitCommandsHelp + "\n\n"

	case showError:
		return m.errorMessage

	default:
		return ""
	}
}

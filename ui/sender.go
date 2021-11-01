package ui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"www.github.com/ZinoKader/portal/tools"
)

const (
	padding         = 2
	maxWidth        = 80
	copyPasswordKey = "c"
)

var quitKeys = []string{"ctrl+c", "q"}

type uiState int

// ui state flows from the top down
const (
	showPasswordWithCopy uiState = iota
	showPassword
	showSendingProgress
)

type senderUIModel struct {
	state       uiState
	fileNames   []string
	payloadSize int64
	password    string
	readyToSend bool
	spinner     spinner.Model
	progressBar progress.Model
}

type FileInfoMsg struct {
	FileNames []string
	Bytes     int64
}

type PasswordMsg struct {
	Password string
}

type ReadyMsg struct{}

type ProgressMsg struct {
	Progress float32
}

var infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(PRIMARY_COLOR)).Render
var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(SECONDARY_COLOR)).Render
var italicText = lipgloss.NewStyle().Italic(true).Render
var boldText = lipgloss.NewStyle().Bold(true).Render

func NewSenderUI() *tea.Program {
	s := spinner.NewModel()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(SPINNER_COLOR))
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

	case FileInfoMsg:
		m.state = showPasswordWithCopy
		m.fileNames = msg.FileNames
		m.payloadSize = msg.Bytes
		return m, nil

	case ReadyMsg:
		m.readyToSend = true
		return m, nil

	case PasswordMsg:
		m.state = showPasswordWithCopy
		m.password = msg.Password
		return m, nil

	case ProgressMsg:
		m.state = showSendingProgress
		if m.progressBar.Percent() == 1.0 {
			return m, tea.Quit
		}
		cmd := m.progressBar.SetPercent(float64(msg.Progress))
		return m, cmd

	case tea.KeyMsg:
		if strings.ToLower(msg.String()) == copyPasswordKey {
			m.state = showPassword
			clipboard.WriteAll(fmt.Sprintf("portal receive %s", m.password))
			return m, nil
		}
		if tools.Contains(quitKeys, strings.ToLower(msg.String())) {
			return m, tea.Quit
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.progressBar.Width = msg.Width - padding*2 - 4
		if m.progressBar.Width > maxWidth {
			m.progressBar.Width = maxWidth
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
	pad := strings.Repeat(" ", padding)

	readiness := fmt.Sprintf("%s Compressing files, preparing", m.spinner.View())
	if m.readyToSend {
		readiness = fmt.Sprintf("%s Awaiting receiver, ready", m.spinner.View())
	}

	fileInfoText := fmt.Sprintf("%s to send file(s)...", readiness)
	if m.fileNames != nil && m.payloadSize != 0 {
		filesToSend := italicText(strings.Join(m.fileNames, ", "))
		payloadSize := boldText(tools.ByteCountSI(m.payloadSize))
		fileInfoText = fmt.Sprintf("%s to send %s (%s)", readiness, filesToSend, payloadSize)
	}

	switch m.state {

	case showPassword, showPasswordWithCopy:
		copyText := "(password copied to clipboard)"
		if m.state == showPasswordWithCopy {
			copyText = "(press 'c' to copy the command to your clipboard)"
		}
		return "\n" +
			pad + infoStyle(fileInfoText) + "\n\n" +
			pad + infoStyle("On the receiving end, run:") + "\n" +
			pad + infoStyle(fmt.Sprintf("portal receive %s", m.password)) + "\n\n" +
			pad + helpStyle(copyText) + "\n\n"

	case showSendingProgress:
		return "\n" +
			pad + infoStyle(fileInfoText) + "\n\n" +
			pad + m.progressBar.View() + "\n\n" +
			pad + helpStyle("Press any key to quit") + "\n\n"

	default:
		return ""
	}
}

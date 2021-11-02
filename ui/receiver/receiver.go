package receiverui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"www.github.com/ZinoKader/portal/tools"
	"www.github.com/ZinoKader/portal/ui"
)

type uiState int

// ui state flows from the top down
const (
	showEstablishing uiState = iota
	showReceivingProgress
	showError
)

type receiverUIModel struct {
	state        uiState
	payloadSize  int64
	spinner      spinner.Model
	progressBar  progress.Model
	errorMessage string
}

func NewReceiverUI() *tea.Program {
	s := spinner.NewModel()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.SPINNER_COLOR))
	m := receiverUIModel{
		spinner:     s,
		progressBar: progress.NewModel(progress.WithDefaultGradient()),
	}
	return tea.NewProgram(m)
}

func (receiverUIModel) Init() tea.Cmd {
	return spinner.Tick
}

func (m receiverUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case ui.FileInfoMsg:
		m.payloadSize = msg.Bytes
		return m, nil

	case ui.ProgressMsg:
		m.state = showReceivingProgress
		if m.progressBar.Percent() == 1.0 {
			return m, tea.Quit
		}
		cmd := m.progressBar.SetPercent(float64(msg.Progress))
		return m, cmd

	case ui.ErrorMsg:
		m.state = showError
		m.errorMessage = msg.Message
		return m, nil

	case tea.KeyMsg:
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

func (m receiverUIModel) View() string {
	pad := strings.Repeat(" ", ui.Padding)

	switch m.state {

	case showEstablishing:
		establishingText := fmt.Sprintf("%s Establishing connection with sender", m.spinner.View())
		return "\n" +
			pad + ui.InfoStyle(establishingText) + "\n\n"

	case showReceivingProgress:
		payloadSize := ui.BoldText(tools.ByteCountSI(m.payloadSize))
		receivingText := fmt.Sprintf("Receiving files (%s)", payloadSize)
		quitCommandsHelp := ui.HelpStyle(fmt.Sprintf("(any of [%s] to abort)", (strings.Join(ui.QuitKeys, ", "))))
		return "\n" +
			pad + ui.InfoStyle(receivingText) + "\n\n" +
			pad + m.progressBar.View() + "\n\n" +
			pad + quitCommandsHelp + "\n\n"

	default:
		return ""
	}
}

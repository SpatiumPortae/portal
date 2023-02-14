package transferprogress

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/SpatiumPortae/portal/ui"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/errors"
)

var Writer io.Writer

type Option func(*Model)

type Model struct {
	PayloadSize                int64
	bytesTransferred           int64
	progress                   float64
	TransferStartTime          *time.Time
	TransferSpeedEstimateBps   int64
	EstimatedRemainingDuration time.Duration

	Width       int
	progressBar progress.Model
}

func Init(program *tea.Program) {
	Writer = &writer{
		program: program,
	}
}

type writer struct {
	program *tea.Program
}

func (w *writer) Write(b []byte) (int, error) {
	w.program.Send(ui.ProgressMsg(len(b)))
	return len(b), nil
}

func (m *Model) StartTransfer() {
	now := time.Now()
	m.TransferStartTime = &now
}

func New(opts ...Option) Model {
	m := Model{
		progressBar: ui.Progressbar,
	}

	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func (Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	return m.progressBar.ViewAs(m.progress)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.Width = msg.Width - 2*ui.PADDING - 4
		if m.Width > ui.MAX_WIDTH {
			m.Width = ui.MAX_WIDTH
		}
		m.progressBar.Width = m.Width
		return m, nil

	case ui.ProgressMsg:
		if m.TransferStartTime == nil {
			now := time.Now()
			m.TransferStartTime = &now
		}
		secondsSpent := time.Since(*m.TransferStartTime).Seconds()
		m.bytesTransferred += int64(msg)
		bytesRemaining := m.PayloadSize - m.bytesTransferred
		linearRemainingSeconds := float64(bytesRemaining) * secondsSpent / float64(m.bytesTransferred)
		if remainingDuration, err := time.ParseDuration(fmt.Sprintf("%fs", linearRemainingSeconds)); err != nil {
			return m, ui.ErrorCmd(errors.Wrap(err, "failed to parse duration of estimated remaining transfer time"))
		} else {
			m.EstimatedRemainingDuration = remainingDuration
		}
		m.TransferSpeedEstimateBps = int64(float64(m.bytesTransferred) / secondsSpent)

		m.progress = math.Min(1.0, float64(m.bytesTransferred)/float64(m.PayloadSize))
		return m, nil

	case progress.FrameMsg:
		progressModel, cmd := m.progressBar.Update(msg)
		m.progressBar = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

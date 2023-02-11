package transferprogress

import (
	"fmt"
	"math"
	"time"

	"github.com/SpatiumPortae/portal/ui"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/errors"
)

type Option func(*Model)

type Model struct {
	PayloadSize                int64
	TransferStartTime          time.Time
	TransferSpeedEstimateBps   int64
	EstimatedRemainingDuration time.Duration

	Width       int
	progress    float64
	progressBar progress.Model
}

func (m *Model) StartTransfer() {
	m.TransferStartTime = time.Now()
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
		secondsSpent := time.Since(m.TransferStartTime).Seconds()
		if m.progress > 0 {
			bytesTransferred := m.progress * float64(m.PayloadSize)
			bytesRemaining := m.PayloadSize - int64(bytesTransferred)
			linearRemainingSeconds := float64(bytesRemaining) * secondsSpent / bytesTransferred
			if remainingDuration, err := time.ParseDuration(fmt.Sprintf("%fs", linearRemainingSeconds)); err != nil {
				return m, ui.ErrorCmd(errors.Wrap(err, "failed to parse duration of estimated remaining transfer time"))
			} else {
				m.EstimatedRemainingDuration = remainingDuration
			}
			m.TransferSpeedEstimateBps = int64(bytesTransferred / secondsSpent)
		}

		currentBytesReceived := float64(msg)
		m.progress = math.Min(1.0, currentBytesReceived/float64(m.PayloadSize))
		return m, nil

	default:
		return m, nil
	}
}

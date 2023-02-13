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
	Width int

	PayloadSize                int64
	bytesTransferred           int64
	TransferStartTime          time.Time
	TransferSpeedEstimateBps   int64
	estimatedRemainingDuration time.Duration

	progress    float64
	progressBar progress.Model
}

func (m *Model) StartTransfer() {
	m.TransferStartTime = time.Now()
}

func New(opts ...Option) Model {
	m := Model{
		progressBar: ui.NewProgressBar(),
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
	bytesProgress := fmt.Sprintf("(%s/%s, %s/s)",
		ui.ByteCountSI(m.bytesTransferred), ui.ByteCountSI(m.PayloadSize), ui.ByteCountSI(m.TransferSpeedEstimateBps))
	eta := fmt.Sprintf("%v remaining", m.estimatedRemainingDuration.Round(time.Second).String())
	progressBar := m.progressBar.ViewAs(m.progress)

	return bytesProgress + "\t\t" + eta + "\n\n" +
		ui.PadText + progressBar
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
		if m.bytesTransferred > 0 {
			bytesRemaining := m.PayloadSize - m.bytesTransferred
			linearRemainingSeconds := float64(bytesRemaining) * secondsSpent / float64(m.bytesTransferred)
			if remainingDuration, err := time.ParseDuration(fmt.Sprintf("%fs", linearRemainingSeconds)); err != nil {
				return m, ui.ErrorCmd(errors.Wrap(err, "failed to parse duration of estimated remaining transfer time"))
			} else {
				m.estimatedRemainingDuration = remainingDuration
			}
			m.TransferSpeedEstimateBps = int64(float64(m.bytesTransferred) / secondsSpent)
		}

		m.bytesTransferred = int64(msg)
		m.progress = math.Min(1.0, float64(m.bytesTransferred)/float64(m.PayloadSize))
		return m, nil

	default:
		return m, nil
	}
}

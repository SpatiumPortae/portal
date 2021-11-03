package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

const (
	PADDING         = 2
	MAX_WIDTH       = 80
	PRIMARY_COLOR   = "#B8BABA"
	SECONDARY_COLOR = "#626262"
	SPINNER_COLOR   = "#9437E9"
	SHUTDOWN_PERIOD = 1 * time.Second
)

var ProgressBar = progress.NewModel(progress.WithGradient("#EE9F70", "#EE9F40"))
var QuitKeys = []string{"ctrl+c", "q", "esc"}
var InfoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(PRIMARY_COLOR)).Render
var HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(SECONDARY_COLOR)).Render
var ItalicText = lipgloss.NewStyle().Italic(true).Render
var BoldText = lipgloss.NewStyle().Bold(true).Render

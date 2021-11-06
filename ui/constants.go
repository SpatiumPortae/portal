package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

const (
	PADDING                 = 2
	MAX_WIDTH               = 80
	PRIMARY_COLOR           = "#B8BABA"
	SECONDARY_COLOR         = "#626262"
	ELEMENT_COLOR           = "#EE9F40"
	SECONDARY_ELEMENT_COLOR = "#EE9F70"
	START_PERIOD            = 1 * time.Millisecond
	SHUTDOWN_PERIOD         = 1000 * time.Millisecond
)

var QuitKeys = []string{"ctrl+c", "q", "esc"}
var PadText = strings.Repeat(" ", PADDING)
var QuitCommandsHelpText = HelpStyle(fmt.Sprintf("(any of [%s] to abort)", (strings.Join(QuitKeys, ", "))))

var ProgressBar = progress.NewModel(progress.WithGradient(SECONDARY_ELEMENT_COLOR, ELEMENT_COLOR))

var baseStyle = lipgloss.NewStyle()
var InfoStyle = baseStyle.Copy().Foreground(lipgloss.Color(PRIMARY_COLOR)).Render
var HelpStyle = baseStyle.Copy().Foreground(lipgloss.Color(SECONDARY_COLOR)).Render
var ItalicText = baseStyle.Copy().Italic(true).Render
var BoldText = baseStyle.Copy().Bold(true).Render

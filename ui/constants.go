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
	ERROR_COLOR             = "#CC0000"
	WARNING_COLOR           = "#FF7900"
	CHECK_COLOR             = "#34B233"
	START_PERIOD            = 1 * time.Millisecond
	SHUTDOWN_PERIOD         = 500 * time.Millisecond
)

var QuitKeys = []string{"ctrl+c", "q", "esc"}
var PadText = strings.Repeat(" ", PADDING)
var QuitCommandsHelpText = HelpStyle(fmt.Sprintf("(any of [%s] to abort)", (strings.Join(QuitKeys, ", "))))

var Progressbar = progress.NewModel(progress.WithGradient(SECONDARY_ELEMENT_COLOR, ELEMENT_COLOR))

var baseStyle = lipgloss.NewStyle()
var InfoStyle = baseStyle.Copy().Foreground(lipgloss.Color(PRIMARY_COLOR)).Render
var HelpStyle = baseStyle.Copy().Foreground(lipgloss.Color(SECONDARY_COLOR)).Render
var ItalicText = baseStyle.Copy().Italic(true).Render
var BoldText = baseStyle.Copy().Bold(true).Render
var ErrorText = baseStyle.Copy().Foreground(lipgloss.Color(ERROR_COLOR)).Render
var WarningText = baseStyle.Copy().Foreground(lipgloss.Color(WARNING_COLOR)).Render
var CheckText = baseStyle.Copy().Foreground(lipgloss.Color(CHECK_COLOR)).Render

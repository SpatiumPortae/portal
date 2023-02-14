package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

const (
	PADDING                  = 2
	MAX_WIDTH                = 80
	PRIMARY_COLOR            = "#B8BABA"
	SECONDARY_COLOR          = "#626262"
	ELEMENT_COLOR            = "#EE9F40"
	SECONDARY_ELEMENT_COLOR  = "#EE9F70"
	ERROR_COLOR              = "#CC6666"
	WARNING_COLOR            = "#EE9F5C"
	CHECK_COLOR              = "#A6E3A1"
	SHUTDOWN_PERIOD          = 500 * time.Millisecond
	TEMP_UI_MESSAGE_DURATION = 2 * time.Second
)

type KeyMap struct {
	Quit         key.Binding
	CopyPassword key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Quit,
		k.CopyPassword,
	}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.CopyPassword},
	}
}

var Keys = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("(q)", "quit"),
	),
	CopyPassword: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("(c)", "copy password to clipboard"),
		key.WithDisabled(),
	),
}

var QuitKeys = []string{"ctrl+c", "q", "esc"}
var PadText = strings.Repeat(" ", PADDING)
var QuitCommandsHelpText = HelpStyle(fmt.Sprintf("(any of [%s] to abort)", strings.Join(QuitKeys, ", ")))

var Progressbar = progress.New(progress.WithGradient(SECONDARY_ELEMENT_COLOR, ELEMENT_COLOR))

var baseStyle = lipgloss.NewStyle()

var InfoStyle = baseStyle.Copy().Foreground(lipgloss.Color(PRIMARY_COLOR)).Render
var HelpStyle = baseStyle.Copy().Foreground(lipgloss.Color(SECONDARY_COLOR)).Render
var ItalicText = baseStyle.Copy().Italic(true).Render
var BoldText = baseStyle.Copy().Bold(true).Render
var ErrorText = baseStyle.Copy().Foreground(lipgloss.Color(ERROR_COLOR)).Render
var WarningText = baseStyle.Copy().Foreground(lipgloss.Color(WARNING_COLOR)).Render
var CheckText = baseStyle.Copy().Foreground(lipgloss.Color(CHECK_COLOR)).Render

var CopyKeyHelpText = baseStyle.Render("copy password to clipboard")
var CopyKeyActiveHelpText = CheckText("✓") + HelpStyle(" copied password to clipboard")

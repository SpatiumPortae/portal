package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

const (
	MARGIN                   = 2
	PADDING                  = 1
	MAX_WIDTH                = 80
	PRIMARY_COLOR            = "#B8BABA"
	SECONDARY_COLOR          = "#626262"
	ELEMENT_COLOR            = "#EE9F40"
	SECONDARY_ELEMENT_COLOR  = "#EE9F70"
	ERROR_COLOR              = "#CC0000"
	WARNING_COLOR            = "#EE9F5C"
	CHECK_COLOR              = "#34B233"
	SHUTDOWN_PERIOD          = 500 * time.Millisecond
	TEMP_UI_MESSAGE_DURATION = 2 * time.Second
)

type KeyMap struct {
	Quit         key.Binding
	CopyPassword key.Binding
	FileListUp   key.Binding
	FileListDown key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Quit,
		k.CopyPassword,
		k.FileListUp,
		k.FileListDown,
	}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.CopyPassword, k.FileListUp, k.FileListDown},
	}
}

func NewProgressBar() progress.Model {
	p := progress.New(progress.WithGradient(SECONDARY_ELEMENT_COLOR, ELEMENT_COLOR))
	p.PercentFormat = "  %.2f%%"
	return p
}

var Keys = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("(q)", "quit"),
	),
	CopyPassword: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("(c)", CopyKeyHelpText),
		key.WithDisabled(),
	),
	FileListUp: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("(↑/k)", "file summary up"),
	),
	FileListDown: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("(↓/j)", "file summary down"),
	),
}

var PadText = strings.Repeat(" ", MARGIN)
var BaseStyle = lipgloss.NewStyle()

var InfoStyle = BaseStyle.Copy().Foreground(lipgloss.Color(PRIMARY_COLOR)).Render
var HelpStyle = BaseStyle.Copy().Foreground(lipgloss.Color(SECONDARY_COLOR)).Render
var ItalicText = BaseStyle.Copy().Italic(true).Render
var BoldText = BaseStyle.Copy().Bold(true).Render
var ErrorText = BaseStyle.Copy().Foreground(lipgloss.Color(ERROR_COLOR)).Render
var WarningText = BaseStyle.Copy().Foreground(lipgloss.Color(WARNING_COLOR)).Render
var CheckText = BaseStyle.Copy().Foreground(lipgloss.Color(CHECK_COLOR)).Render

var CopyKeyHelpText = BaseStyle.Render("password → clipboard")
var CopyKeyActiveHelpText = CheckText("✓") + HelpStyle(" password → clipboard")

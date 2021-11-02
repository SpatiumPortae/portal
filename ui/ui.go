package ui

import "github.com/charmbracelet/lipgloss"

const (
	Padding  = 2
	MaxWidth = 80
)

type FileInfoMsg struct {
	FileNames []string
	Bytes     int64
}

type ErrorMsg struct {
	Message string
}

type ProgressMsg struct {
	Progress float32
}

var QuitKeys = []string{"ctrl+c", "q", "esc"}

var InfoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(PRIMARY_COLOR)).Render
var HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(SECONDARY_COLOR)).Render
var ItalicText = lipgloss.NewStyle().Italic(true).Render
var BoldText = lipgloss.NewStyle().Bold(true).Render

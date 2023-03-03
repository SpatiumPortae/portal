package filetable

import (
	"math"

	"github.com/SpatiumPortae/portal/cmd/portal/tui"
	"github.com/SpatiumPortae/portal/internal/file"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const (
	defaultMaxTableHeight         = 4
	nameColumnWidthFactor float64 = 0.8
	sizeColumnWidthFactor float64 = 1 - nameColumnWidthFactor
)

var fileTableStyle = tui.BaseStyle.Copy().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(tui.SECONDARY_COLOR)).
	MarginLeft(tui.MARGIN)

type Option func(m *Model)

type fileRow struct {
	path          string
	formattedSize string
}

type Model struct {
	Width       int
	MaxHeight   int
	rows        []fileRow
	table       table.Model
	tableStyles table.Styles
}

func New(opts ...Option) Model {
	m := Model{
		MaxHeight: defaultMaxTableHeight,
		table: table.New(
			table.WithFocused(true),
			table.WithHeight(defaultMaxTableHeight),
		),
	}

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(tui.SECONDARY_COLOR)).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(tui.DARK_COLOR)).
		Background(lipgloss.Color(tui.SECONDARY_ELEMENT_COLOR)).
		Bold(false)
	m.tableStyles = s
	m.table.SetStyles(m.tableStyles)

	m.updateColumns()
	for _, opt := range opts {
		opt(&m)
	}

	return m
}

func (m *Model) SetFiles(filePaths []string) {
	for _, filePath := range filePaths {
		size, err := file.FileSize(filePath)
		var formattedSize string
		if err != nil {
			formattedSize = "N/A"
		} else {
			formattedSize = tui.ByteCountSI(size)
		}
		m.rows = append(m.rows, fileRow{path: filePath, formattedSize: formattedSize})
	}
	m.table.SetHeight(int(math.Min(float64(m.MaxHeight), float64(len(filePaths)))))
	m.updateColumns()
	m.updateRows()
}

func WithFiles(filePaths []string) Option {
	return func(m *Model) {
		m.SetFiles(filePaths)
	}
}

func (m *Model) SetMaxHeight(height int) {
	m.MaxHeight = height
}

func WithMaxHeight(height int) Option {
	return func(m *Model) {
		m.SetMaxHeight(height)
		m.updateRows()
	}
}

func (m *Model) getMaxWidth() int {
	return int(math.Min(tui.MAX_WIDTH-2*tui.MARGIN, float64(m.Width)))
}

func (m *Model) updateColumns() {
	w := m.getMaxWidth()
	m.table.SetColumns([]table.Column{
		{Title: "File", Width: int(float64(w) * nameColumnWidthFactor)},
		{Title: "Size", Width: int(float64(w) * sizeColumnWidthFactor)},
	})
}

func (m *Model) updateRows() {
	var tableRows []table.Row
	maxFilePathWidth := int(float64(m.getMaxWidth()) * nameColumnWidthFactor)
	for _, row := range m.rows {
		path := row.path
		// truncate overflowing file paths from the left
		if len(path) > maxFilePathWidth {
			overflowingLength := len(path) - maxFilePathWidth
			path = runewidth.TruncateLeft(path, overflowingLength+1, "…")
		}
		tableRows = append(tableRows, table.Row{path, row.formattedSize})
	}
	m.table.SetRows(tableRows)
}

func (Model) Init() tea.Cmd {
	return nil
}

func (m Model) Finalize() tea.Model {
	m.table.Blur()

	s := m.tableStyles
	s.Selected = s.Selected.UnsetBackground().UnsetForeground()
	m.table.SetStyles(s)

	return m
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.Width = msg.Width - 2*tui.MARGIN - 4
		if m.Width > tui.MAX_WIDTH {
			m.Width = tui.MAX_WIDTH
		}
		m.updateColumns()
		m.updateRows()
		return m, nil

	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if len(m.rows) == 0 {
		return ""
	}
	return fileTableStyle.Render(m.table.View()) + "\n\n"
}

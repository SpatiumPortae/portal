package filetable

import (
	"math"

	"github.com/SpatiumPortae/portal/internal/file"
	"github.com/SpatiumPortae/portal/ui"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const (
	maxTableHeight                = 4
	nameColumnWidthFactor float64 = 0.8
	sizeColumnWidthFactor float64 = 1 - nameColumnWidthFactor
)

var fileTableStyle = ui.BaseStyle.Copy().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(ui.SECONDARY_COLOR)).
	MarginLeft(ui.MARGIN)

type Option func(m *Model)

type fileRow struct {
	path          string
	formattedSize string
}

type Model struct {
	Width int
	rows  []fileRow
	table table.Model
}

func New(opts ...Option) Model {
	m := Model{
		table: table.New(
			table.WithFocused(true),
			table.WithHeight(maxTableHeight),
		),
	}

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	m.table.SetStyles(s)

	m.updateColumns()
	for _, opt := range opts {
		opt(&m)
	}

	return m
}

func WithFiles(filePaths []string) Option {
	return func(m *Model) {
		for _, filePath := range filePaths {
			size, err := file.FileSize(filePath)
			var formattedSize string
			if err != nil {
				formattedSize = "N/A"
			} else {
				formattedSize = ui.ByteCountSI(size)
			}
			m.rows = append(m.rows, fileRow{path: filePath, formattedSize: formattedSize})
		}
		m.table.SetHeight(int(math.Min(maxTableHeight, float64(len(filePaths)))))
		m.updateColumns()
		m.updateRows()
	}
}

func (m *Model) getMaxWidth() int {
	return int(math.Min(ui.MAX_WIDTH-2*ui.MARGIN, float64(m.Width)))
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.Width = msg.Width - 2*ui.MARGIN - 4
		if m.Width > ui.MAX_WIDTH {
			m.Width = ui.MAX_WIDTH
		}
		m.updateColumns()
		m.updateRows()
		return m, nil

	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return fileTableStyle.Render(m.table.View()) + "\n\n"
}

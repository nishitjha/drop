package internal

import (
	"fmt"
	"os"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var BaseStyle = lipgloss.NewStyle().Margin(1, 0)

type Model struct {
	Table table.Model
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}

	m.Table, cmd = m.Table.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	return tea.NewView(BaseStyle.Render(m.Table.View()) + "\n")
}

func RenderTable(entries []table.Row) {
	columns := []table.Column{
		{Title: "Device Name", Width: 20},
		{Title: "Status", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(entries),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()

	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(true).
		Align(lipgloss.Left).
		Padding(0, 1)

	s.Cell = s.Cell.
		Align(lipgloss.Left).
		Padding(0, 1)

	s.Selected = s.Cell

	t.SetStyles(s)

	t.SetStyles(s)

	m := Model{Table: t}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

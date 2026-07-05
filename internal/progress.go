package internal

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	padding  = 2
	maxWidth = 80
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

func Launch() {
	m := ProgressModel{
		Progress: progress.New(progress.WithDefaultBlend()),
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
	}
}

type progressMsg struct {
	Decimal float64
}
type doneMsg struct {
	Err error
}

type ProgressModel struct {
	Progress progress.Model
	Err      error
}

func (m ProgressModel) Init() tea.Cmd {
	return nil
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.Progress.SetWidth(msg.Width - padding*2 - 4)
		if m.Progress.Width() > maxWidth {
			m.Progress.SetWidth(maxWidth)
		}
		return m, nil

	case progressMsg:
		if m.Progress.Percent() == 1.0 {
			return m, tea.Quit
		}

		cmd := m.Progress.IncrPercent(msg.Decimal)
		return m, cmd
	case doneMsg:
		m.Err = msg.Err
		return m, tea.Quit
		
	case progress.FrameMsg:
		var cmd tea.Cmd
		m.Progress, cmd = m.Progress.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

func (m ProgressModel) View() tea.View {
	pad := strings.Repeat(" ", padding)
	return tea.NewView("\n" +
		pad + m.Progress.View() + "\n\n" +
		pad + helpStyle("Press any key to quit"))
}

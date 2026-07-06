package internal

import (
	"fmt"
	"os"
	"path/filepath"
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

func Launch(deviceAddress string, deviceName string, filePath string) {
	m := ProgressModel{
		Progress:      progress.New(progress.WithDefaultBlend()),
		DeviceAddress: deviceAddress,
		DeviceName:    deviceName,
		FilePath:      filePath,
	}

	p := tea.NewProgram(m)

	go StreamFile(deviceAddress, deviceName, filePath, p)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
	}

	fm := finalModel.(ProgressModel)
	for _, l := range fm.Log {
		fmt.Println(l)
	}
}

type progressMsg struct {
	Decimal float64
}

type doneMsg struct {
	Err     error
	Message string
}

type statusMsg string
type logMsg string

type ProgressModel struct {
	Progress      progress.Model
	Err           error
	DeviceAddress string
	DeviceName    string
	FilePath      string
	Status        string
	Log           []string
	Done          bool
}

func (m ProgressModel) Init() tea.Cmd {
	return nil
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			if !m.Done {
				m.Log = append(m.Log, fmt.Sprintf("%s Stopped sharing \"%s\" with \"%s\".", Icons.Negative, filepath.Base(m.FilePath), m.DeviceName))
			}
			m.Done = true
			return m, tea.Quit
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.Progress.SetWidth(msg.Width - padding*2 - 4)
		if m.Progress.Width() > maxWidth {
			m.Progress.SetWidth(maxWidth)
		}
		return m, nil
	case statusMsg:
		m.Status = string(msg)
		return m, nil
	case logMsg:
		m.Log = append(m.Log, string(msg))
		return m, nil
	case progressMsg:
		cmd := m.Progress.SetPercent(msg.Decimal)
		return m, cmd
	case doneMsg:
		if msg.Err != nil {
			m.Log = append(m.Log, fmt.Sprintf("%s %v", Icons.Negative, msg.Err))
		} else if msg.Message != "" {
			m.Log = append(m.Log, msg.Message)
		}
		m.Err = msg.Err
		m.Done = true
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
	s := "\n"
	if m.Status != "" {
		s += pad + m.Status + "\n\n"
	}
	s += pad + m.Progress.View() + "\n\n"
	for _, l := range m.Log {
		s += pad + l + "\n"
	}
	if !m.Done {
		s += pad + helpStyle("Press ctrl + c, q, or esc to quit")
	}
	view := tea.NewView(s)
	view.AltScreen = true
	return view
}
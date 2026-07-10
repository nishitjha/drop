package internal

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
)

type FileModel struct {
	Filepicker   filepicker.Model
	SelectedFile string
	Quitting     bool
	Err          error
	DirMode      bool
}

type ClearErrorMsg struct{}

func ClearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return ClearErrorMsg{}
	})
}

func (m FileModel) Init() tea.Cmd {
	return m.Filepicker.Init()
}

func (m FileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit
		}
	case ClearErrorMsg:
		m.Err = nil
	}

	var cmd tea.Cmd
	m.Filepicker, cmd = m.Filepicker.Update(msg)

	if didSelect, path := m.Filepicker.DidSelectFile(msg); didSelect {
		m.SelectedFile = path
		return m, tea.Quit
	}

	return m, cmd
}

func (m FileModel) View() tea.View {
	if m.Quitting {
		return tea.NewView("")
	}

	var s strings.Builder
	if !m.DirMode {
		s.WriteString("\nPick a " + m.Filepicker.Styles.File.Render("file") + " for sharing. Use the arrow keys or your mouse wheel to scroll and navigate. \nPress enter to choose the selected file.\n\n")
	} else {
		s.WriteString("\nPick a " + m.Filepicker.Styles.File.Render("folder") + " for sharing. Use the arrow keys or your mouse wheel to scroll and navigate. \nPress enter to choose the selected folder.\n\n")
	}

	if m.Err != nil {
		s.WriteString(m.Filepicker.Styles.DisabledFile.Render(m.Err.Error()))
	} else if m.SelectedFile == "" {
		if !m.DirMode {
			s.WriteString("Pick a file:")
		} else {
			s.WriteString("Pick a folder:")
		}
	} else {
		s.WriteString("Selected " + (func() string {
			if !m.DirMode {
				return "file"
			}
			return "folder"
		})() + ": " + m.Filepicker.Styles.Selected.Render(m.SelectedFile))
	}

	s.WriteString("\n\n" + m.Filepicker.View() + "\n")

	v := tea.NewView(s.String())
	v.AltScreen = true
	return v
}
package internal

import (
	"fmt"
	"net/http"
	"os"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

type TaskResultMsg struct {
	Response *http.Response
	Error      error
}

type SpinnerModel struct {
	Spinner spinner.Model
	Text    string
	Task    func() tea.Msg
	Result  TaskResultMsg
}

var _ tea.Model = SpinnerModel{}

func (m SpinnerModel) Init() tea.Cmd {
	return tea.Batch(m.Spinner.Tick, m.Task)
}

func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg: 
		if msg.String() == "ctrl+c" {
			os.Exit(0)
		}
	case TaskResultMsg:
		m.Result = msg
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m SpinnerModel) View() tea.View {
	str := fmt.Sprintf("\n %s %s\n", m.Spinner.View(), m.Text)
	return tea.NewView(str)
}

func RunSpinner(text string, task func() tea.Msg) TaskResultMsg {
	s := spinner.New()
	s.Spinner = spinner.Monkey

	m := SpinnerModel{
		Spinner: s,
		Text:    text,
		Task:    task,
	}

	p := tea.NewProgram(m)
	finalModel, _ := p.Run()
	return finalModel.(SpinnerModel).Result
}
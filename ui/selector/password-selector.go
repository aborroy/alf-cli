package selector

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type passwordModel struct {
	input    textinput.Model
	quitting bool
}

func (m passwordModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m passwordModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		// Optionally handle window size
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m passwordModel) View() string {
	if m.quitting {
		return ""
	}
	return passwordPromptStyle.Render(m.input.View()) + "\n"
}

func RunPasswordInput(prompt string) (string, error) {
	for {
		ti := textinput.New()
		ti.Placeholder = ""
		ti.Focus()
		ti.CharLimit = 256
		ti.Width = 40
		ti.PromptStyle = titleStyle
		ti.Prompt = prompt + ": "
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'

		p := tea.NewProgram(passwordModel{input: ti})
		m, err := p.Run()
		if err != nil {
			return "", err
		}

		value := m.(passwordModel).input.Value()
		if value != "" {
			return value, nil
		}

		prompt = "\tPassword (cannot be empty)"
	}
}

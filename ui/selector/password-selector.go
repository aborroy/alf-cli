package selector

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type passwordModel struct {
	input          textinput.Model
	quitting       bool
	defaultCleared bool
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

		if !m.defaultCleared {
			// Only process printable characters
			if len(msg.String()) == 1 && msg.Type == tea.KeyRunes {
				// Clear input and switch to password mode
				m.input.SetValue("")
				m.input.EchoMode = textinput.EchoPassword
				m.input.EchoCharacter = '•'
				m.defaultCleared = true

				// Re-send the same key as new input
				// (this ensures only the typed char is inserted)
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
			// Ignore other non-character keys until the user types a letter
			return m, nil
		}
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

func RunPasswordInput(prompt, defaultValue string) (string, error) {
	for {
		ti := textinput.New()
		ti.Placeholder = ""
		ti.Focus()
		ti.CharLimit = 256
		ti.Width = 40
		ti.PromptStyle = titleStyle
		ti.Prompt = prompt + ": "

		if defaultValue != "" {
			ti.SetValue(defaultValue)
			ti.EchoMode = textinput.EchoNormal // show the default in clear
		} else {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '•'
		}

		p := tea.NewProgram(passwordModel{input: ti})
		m, err := p.Run()
		if err != nil {
			return "", err
		}

		value := m.(passwordModel).input.Value()
		if value != "" {
			return value, nil
		}
		if defaultValue != "" {
			return defaultValue, nil
		}

		prompt = "\tPassword (cannot be empty)"
	}
}

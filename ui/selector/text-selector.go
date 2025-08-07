package selector

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type inputModel struct {
	input    textinput.Model
	quitting bool
}

func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	return inputPromptStyle.Render(m.input.View()) + "\n"
}

func RunTextInput(prompt string, defaultValue string) (string, error) {
	ti := textinput.New()
	ti.Placeholder = defaultValue
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40
	ti.PromptStyle = titleStyle
	ti.Prompt = prompt + ": "

	p := tea.NewProgram(inputModel{input: ti})
	m, err := p.Run()
	if err != nil {
		return "", err
	}

	value := m.(inputModel).input.Value()
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

package selector

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// RunTextInput prompts for a line of text with a default.
// Final frame prints "<prompt>: <value>" and stays in the transcript.
func RunTextInput(prompt, defaultValue string) (string, error) {
	ti := textinput.New()
	ti.Focus()
	ti.SetValue(defaultValue) // pressing Enter accepts default
	ti.Prompt = prompt + ": "
	ti.CharLimit = 0

	m := inputModel{
		prompt: prompt,
		input:  ti,
		def:    defaultValue,
		dirty:  false,
	}

	p := tea.NewProgram(m)
	res, err := p.Run()
	if err != nil {
		return "", err
	}
	final := res.(inputModel)

	v := final.input.Value()
	if v == "" {
		v = defaultValue
	}
	return v, nil
}

type inputModel struct {
	prompt string
	input  textinput.Model
	def    string
	dirty  bool
	done   bool
	final  string
}

func (m inputModel) Init() tea.Cmd { return textinput.Blink }

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		case "enter":
			val := m.input.Value()
			if val == "" {
				val = m.def
			}
			m.final = fmt.Sprintf("%s: %s", m.prompt, val)
			m.done = true
			return m, tea.Quit
		default:
			// First printable char wipes the prefilled default so typing replaces it.
			if !m.dirty && len(msg.String()) == 1 {
				m.input.SetValue("")
				m.dirty = true
			}
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	if m.done {
		return m.final + "\n"
	}
	return m.input.View()
}

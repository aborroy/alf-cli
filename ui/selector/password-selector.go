package selector

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// RunPasswordInput prompts for a password (masked). If default is provided and user
// presses Enter, we accept it; final frame prints "<prompt>: ********".
func RunPasswordInput(prompt, defaultValue string) (string, error) {
	ti := textinput.New()
	ti.Focus()
	ti.Prompt = prompt + ": "
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.SetValue(defaultValue) // Enter accepts default; masked during edit

	m := passwordModel{
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
	final := res.(passwordModel)

	v := final.input.Value()
	if v == "" {
		v = defaultValue
	}
	return v, nil
}

type passwordModel struct {
	prompt string
	input  textinput.Model
	def    string
	dirty  bool
	done   bool
	final  string
}

func (m passwordModel) Init() tea.Cmd { return textinput.Blink }

func (m passwordModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.final = fmt.Sprintf("%s: %s", m.prompt, strings.Repeat("•", len(val)))
			m.done = true
			return m, tea.Quit
		default:
			// First printable char clears default so you start fresh.
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

func (m passwordModel) View() string {
	if m.done {
		return m.final + "\n"
	}
	return m.input.View()
}

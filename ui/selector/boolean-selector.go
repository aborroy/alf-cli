package selector

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type compactYNModel struct {
	title      string
	options    []string
	selected   int
	choice     bool
	quitting   bool
	defaultIdx int
	width      int
}

func (m compactYNModel) Init() tea.Cmd {
	return nil
}

func (m compactYNModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "left", "h":
			if m.selected > 0 {
				m.selected--
			}
		case "right", "l":
			if m.selected < len(m.options)-1 {
				m.selected++
			}
		case "enter":
			m.choice = (m.selected == 0)
			return m, tea.Quit
		case "esc":
			m.selected = m.defaultIdx
			m.choice = (m.defaultIdx == 0)
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m compactYNModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(m.title))
	b.WriteString(" ")

	for i, option := range m.options {
		if i == m.selected {
			formatted := fmt.Sprintf("[%s]", option)
			if i == m.defaultIdx {
				formatted = fmt.Sprintf("[%s*]", option)
			}
			b.WriteString(selectedBooleanStyle.Render(formatted))
		} else {
			formatted := option
			if i == m.defaultIdx {
				formatted = fmt.Sprintf("%s*", option)
			}
			b.WriteString(booleanStyle.Render(formatted))
		}

		if i < len(m.options)-1 {
			b.WriteString(booleanStyle.Render(" / "))
		}
	}

	return lipgloss.NewStyle().Width(m.width).Render(b.String()) + "\n"
}

func RunYesNoSelector(prompt string, defaultValue bool) (bool, error) {
	defaultIdx := 0
	if !defaultValue {
		defaultIdx = 1
	}

	m := compactYNModel{
		title:      prompt,
		options:    []string{"Yes", "No"},
		selected:   defaultIdx,
		defaultIdx: defaultIdx,
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return false, err
	}

	final := result.(compactYNModel)
	return final.choice, nil
}

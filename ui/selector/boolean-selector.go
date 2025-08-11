package selector

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// RunYesNoSelector shows a compact Yes/No selector that keeps a running transcript.
func RunYesNoSelector(prompt string, defaultYes bool) (bool, error) {
	defaultIdx := 0
	if !defaultYes {
		defaultIdx = 1
	}

	m := compactYNModel{
		title:      prompt,
		options:    []string{"Yes", "No"},
		selected:   defaultIdx,
		defaultIdx: defaultIdx,
	}

	p := tea.NewProgram(m)
	res, err := p.Run()
	if err != nil {
		return false, err
	}
	final := res.(compactYNModel)
	return final.choice, nil
}

type compactYNModel struct {
	title      string
	options    []string
	selected   int
	defaultIdx int
	choice     bool
	done       bool
}

func (m compactYNModel) Init() tea.Cmd { return nil }

func (m compactYNModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.options)-1 {
				m.selected++
			}
		case "enter":
			m.choice = (m.selected == 0)
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m compactYNModel) View() string {
	if m.done {
		ans := "No"
		if m.choice {
			ans = "Yes"
		}
		return fmt.Sprintf("%s: %s\n", m.title, ans)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", m.title)
	for i, opt := range m.options {
		cursor := "  "
		if i == m.selected {
			cursor = "> "
		}
		fmt.Fprintf(&b, "%s%s\n", cursor, opt)
	}
	fmt.Fprintln(&b, "\n↑/↓ to move • Enter to select")
	return b.String()
}

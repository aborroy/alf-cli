package selector

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Option struct {
	Code        string
	Description string
}

func (o Option) String() string {
	if o.Description != "" {
		return fmt.Sprintf("%s - %s", o.Code, o.Description)
	}
	return o.Code
}

type compactModel struct {
	title         string
	options       []Option
	selected      int
	selectedItems map[int]bool
	multiSelect   bool
	choices       []Option
	quitting      bool
}

func (m compactModel) Init() tea.Cmd {
	return nil
}

func (m compactModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
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
			if m.multiSelect {
				if m.selectedItems[m.selected] {
					delete(m.selectedItems, m.selected)
				} else {
					m.selectedItems[m.selected] = true
				}
			} else {
				m.choices = []Option{m.options[m.selected]}
				return m, tea.Quit
			}
		case " ", "space":
			if m.multiSelect {
				if m.selectedItems[m.selected] {
					delete(m.selectedItems, m.selected)
				} else {
					m.selectedItems[m.selected] = true
				}
			}
		case "ctrl+a":
			if m.multiSelect {
				for i := range m.options {
					m.selectedItems[i] = true
				}
			}
		case "ctrl+d":
			if m.multiSelect {
				m.selectedItems = make(map[int]bool)
			}
		case "tab", "ctrl+m":
			if m.multiSelect {
				var selected []Option
				for i := range m.options {
					if m.selectedItems[i] {
						selected = append(selected, m.options[i])
					}
				}
				m.choices = selected
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m compactModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(m.title))

	if m.multiSelect {
		b.WriteString("\n")
		help := "Space/Enter: toggle, Tab: finish, Ctrl+A: select all, Ctrl+D: deselect all"
		b.WriteString(helpStyle.Render(help))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}

	for i, option := range m.options {
		var line string
		prefix := " "

		if m.multiSelect {
			if m.selectedItems[i] {
				prefix = "✓"
			} else {
				prefix = "○"
			}
		}

		if i == m.selected {
			if m.multiSelect && m.selectedItems[i] {
				line = selectedItemStyle.Render(fmt.Sprintf("%s %d. %s",
					checkedStyle.Render(prefix), i+1, option.String()))
			} else {
				line = selectedItemStyle.Render(fmt.Sprintf("> %s %d. %s",
					prefix, i+1, option.String()))
			}
		} else {
			if m.multiSelect && m.selectedItems[i] {
				line = checkedStyle.Render(fmt.Sprintf("  %s %d. %s",
					prefix, i+1, option.String()))
			} else {
				line = itemStyle.Render(fmt.Sprintf("  %s %d. %s",
					prefix, i+1, option.String()))
			}
		}

		b.WriteString(line)
		if i < len(m.options)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func RunSelector(prompt string, opts []string) (string, error) {
	options := make([]Option, len(opts))
	for i, opt := range opts {
		options[i] = Option{Code: opt}
	}

	selected, err := RunSelectorWithOptions(prompt, options, false)
	if err != nil {
		return "", err
	}

	if len(selected) > 0 {
		return selected[0].Code, nil
	}
	return "", nil
}

func RunSelectorWithOptions(prompt string, opts []Option, multiSelect bool) ([]Option, error) {
	m := compactModel{
		title:         prompt,
		options:       opts,
		multiSelect:   multiSelect,
		selectedItems: make(map[int]bool),
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	final := result.(compactModel)
	return final.choices, nil
}

func RunMultiSelector(prompt string, opts []string) ([]string, error) {
	options := make([]Option, len(opts))
	for i, opt := range opts {
		options[i] = Option{Code: opt}
	}

	selected, err := RunSelectorWithOptions(prompt, options, true)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(selected))
	for i, opt := range selected {
		result[i] = opt.Code
	}
	return result, nil
}

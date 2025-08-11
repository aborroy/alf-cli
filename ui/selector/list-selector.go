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
		return fmt.Sprintf("%s — %s", o.Code, o.Description)
	}
	return o.Code
}

// RunSelector: single-select over []string; final frame prints "✔ <value>".
func RunSelector(prompt string, opts []string) (string, error) {
	options := make([]Option, len(opts))
	for i, s := range opts {
		options[i] = Option{Code: s}
	}
	selected, err := RunSelectorWithOptions(prompt, options, false)
	if err != nil {
		return "", err
	}
	return selected[0].Code, nil
}

// RunMultiSelector: multi-select over []string; final frame prints "✔ a, b, c".
func RunMultiSelector(prompt string, opts []string) ([]string, error) {
	options := make([]Option, len(opts))
	for i, s := range opts {
		options[i] = Option{Code: s}
	}
	selected, err := RunSelectorWithOptions(prompt, options, true)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(selected))
	for i, o := range selected {
		out[i] = o.Code
	}
	return out, nil
}

// RunSelectorWithOptions: core single/multi implementation with transcript-preserving summary.
func RunSelectorWithOptions(prompt string, opts []Option, multiSelect bool) ([]Option, error) {
	m := listModel{
		prompt:      prompt,
		options:     opts,
		selected:    0,
		multi:       multiSelect,
		picked:      make(map[int]bool),
		lineHelp:    "",
		visibleSize: 12,
	}
	if multiSelect {
		m.lineHelp = "Space: toggle • Enter: finish • Ctrl+A: all • Ctrl+D: none"
	} else {
		m.lineHelp = "↑/↓ to move • Enter to select"
	}

	p := tea.NewProgram(m)
	res, err := p.Run()
	if err != nil {
		return nil, err
	}
	final := res.(listModel)

	if final.multi {
		var out []Option
		for i := range final.options {
			if final.picked[i] {
				out = append(out, final.options[i])
			}
		}
		return out, nil
	}
	return []Option{final.options[final.selected]}, nil
}

type listModel struct {
	prompt      string
	options     []Option
	selected    int
	multi       bool
	picked      map[int]bool
	done        bool
	summary     string
	lineHelp    string
	visibleSize int
	offset      int
}

func (m listModel) Init() tea.Cmd { return nil }

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case " ", "space":
			if m.multi {
				m.picked[m.selected] = !m.picked[m.selected]
			}
		case "ctrl+a":
			if m.multi {
				for i := range m.options {
					m.picked[i] = true
				}
			}
		case "ctrl+d":
			if m.multi {
				m.picked = make(map[int]bool)
			}
		case "enter":
			if m.multi {
				var vals []string
				for i := range m.options {
					if m.picked[i] {
						vals = append(vals, m.options[i].Code)
					}
				}
				valsText := "(none)"
				if len(vals) > 0 {
					valsText = strings.Join(vals, ", ")
				}
				m.summary = fmt.Sprintf("%s: %s", m.prompt, valsText)
			} else {
				m.summary = fmt.Sprintf("%s: %s", m.prompt, m.options[m.selected].Code)
			}
			m.done = true
			return m, tea.Quit
		}
	}

	// Simple scroll window
	if m.selected < m.offset {
		m.offset = m.selected
	}
	if m.selected >= m.offset+m.visibleSize {
		m.offset = m.selected - m.visibleSize + 1
	}
	return m, nil
}

func (m listModel) View() string {
	if m.done {
		return m.summary + "\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", m.prompt)

	start := m.offset
	end := start + m.visibleSize
	if end > len(m.options) {
		end = len(m.options)
	}

	for i := start; i < end; i++ {
		cursor := "  "
		if i == m.selected {
			cursor = "> "
		}
		if m.multi {
			box := "[ ] "
			if m.picked[i] {
				box = "[x] "
			}
			fmt.Fprintf(&b, "%s%s%s\n", cursor, box, m.options[i].String())
		} else {
			fmt.Fprintf(&b, "%s%s\n", cursor, m.options[i].String())
		}
	}

	if end < len(m.options) {
		fmt.Fprintf(&b, "  … %d more\n", len(m.options)-end)
	}

	if m.lineHelp != "" {
		fmt.Fprintf(&b, "\n%s\n", m.lineHelp)
	}
	return b.String()
}

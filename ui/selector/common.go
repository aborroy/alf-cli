package selector

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// Alfresco Colors: Blue: #017dff, Yellow: #fba100, Green: #8bdc01

var (
	titleStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("white")).Bold(true)
	itemStyle            = lipgloss.NewStyle().PaddingLeft(0)
	selectedItemStyle    = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("#fba100"))
	booleanStyle         = lipgloss.NewStyle().PaddingLeft(0)
	selectedBooleanStyle = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("#fba100"))
	checkedStyle         = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("#8bdc01"))
	helpStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	paginationStyle      = list.DefaultStyles().PaginationStyle.PaddingLeft(0)
	inputPromptStyle     = lipgloss.NewStyle().MarginLeft(0)
	passwordPromptStyle  = lipgloss.NewStyle().MarginLeft(0)
)

func ApplyListDefaults(l *list.Model) {
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
}

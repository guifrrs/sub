package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	input textinput.Model
}

func NewModel() Model {
	input := textinput.New()
	input.Focus()
	input.Placeholder = "Search subtitles..."
	input.CharLimit = 120
	input.Width = 40

	return Model{input: input}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString("sub - bootstrap\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\nPress esc or ctrl+c to quit.\n")
	return b.String()
}

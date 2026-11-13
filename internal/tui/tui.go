package tui

import tea "github.com/charmbracelet/bubbletea"

type model struct{}

func (model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyRunes:
			if string(keyMsg.Runes) == "q" {
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (model) View() string {
	return "ts4ccmanager\n\nPlaceholder TUI\n\nPress q or esc to quit."
}

// Run starts the placeholder terminal user interface.
func Run() error {
	_, err := tea.NewProgram(model{}, tea.WithAltScreen()).Run()
	return err
}

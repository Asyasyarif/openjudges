package root

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Update handles events
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
		case "esc":
			if m.Mode == ModeCommand {
				m.Mode = ModeBlank
				m.Input.SetValue("")
				m.FilterSuggestions("")
				m.Input.Blur()
				return m, nil
			}
		}

		if m.Mode == ModeBlank {
			// Start typing anywhere to enter command mode
			if msg.Type == tea.KeyRunes || msg.String() == "/" {
				m.Mode = ModeCommand
				m.Input.Focus()
				m.Input, cmd = m.Input.Update(msg)
				m.FilterSuggestions(m.Input.Value())
				return m, cmd
			}
		} else {
			// Command Mode
			switch msg.String() {
			case "enter":
				// Prefer selected item if there are filtered options and we haven't typed a full command yet?
				// Actually, simplify: if there are filtered items, use the selected one.
				// Unless the user typed something that isn't in the list?

				selectedCmd := m.Input.Value()
				if len(m.Filtered) > 0 {
					selectedCmd = m.Filtered[m.Selected].Name
				} else if selectedCmd == "" {
					// No command
					return m, nil
				}

				// Add slash if missing
				if !strings.HasPrefix(selectedCmd, "/") {
					// suggestions don't have slash in my model init, wait:
					// In model.go, suggestions are "create", "run" etc.
					// When we execute, we expect "/create" or just "create"?
					// The legacy system expects "/create".
				}

				m.ExecutedCmd = "/" + strings.TrimPrefix(selectedCmd, "/")
				m.Quitting = true
				return m, tea.Quit

			case "up":
				if m.Selected > 0 {
					m.Selected--
				}
				return m, nil
			case "down":
				if m.Selected < len(m.Filtered)-1 {
					m.Selected++
				}
				return m, nil
			case "tab":
				if len(m.Filtered) > 0 {
					val := m.Filtered[m.Selected].Name
					if strings.HasPrefix(m.Input.Value(), "/") {
						val = "/" + strings.TrimPrefix(val, "/")
					}
					m.Input.SetValue(val)
					m.Input.CursorEnd()
					m.FilterSuggestions(m.Input.Value())
				}
				return m, nil
			}

			// Handle text input
			m.Input, cmd = m.Input.Update(msg)
			if m.Input.Value() == "" {
				m.Mode = ModeBlank
				m.Input.Blur()
			}
			m.FilterSuggestions(m.Input.Value())
			return m, cmd
		}
	}

	return m, nil
}

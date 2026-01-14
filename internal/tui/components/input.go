package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"openjudges/internal/styles"
)

// Input is a reusable text input component
type Input struct {
	textinput textinput.Model
	label     string
	help      string
	required  bool
	validator func(string) error
	err       error
	focused   bool
}

// NewInput creates a new input component
func NewInput(label, placeholder string) Input {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 256
	ti.Width = 50

	return Input{
		textinput: ti,
		label:     label,
		focused:   false,
	}
}

// NewInputRequired creates a new required input component
func NewInputRequired(label, placeholder string) Input {
	input := NewInput(label, placeholder)
	input.required = true
	input.validator = func(s string) error {
		if s == "" {
			return fmt.Errorf("this field is required")
		}
		return nil
	}
	return input
}

// WithHelp adds help text to the input
func (i Input) WithHelp(help string) Input {
	i.help = help
	return i
}

// WithValidator adds a custom validator
func (i Input) WithValidator(validator func(string) error) Input {
	i.validator = validator
	return i
}

// SetWidth sets the input width
func (i *Input) SetWidth(width int) {
	i.textinput.Width = width
}

// WithWidth sets the input width
func (i Input) WithWidth(width int) Input {
	i.textinput.Width = width
	return i
}

// WithDefaultValue sets a default value
func (i Input) WithDefaultValue(value string) Input {
	i.textinput.SetValue(value)
	return i
}

// Focus focuses the input
func (i *Input) Focus() tea.Cmd {
	i.focused = true
	return i.textinput.Focus()
}

// Blur blurs the input
func (i *Input) Blur() {
	i.focused = false
	i.textinput.Blur()
}

// Value returns the current input value
func (i Input) Value() string {
	return i.textinput.Value()
}

// SetValue sets the input value
func (i *Input) SetValue(value string) {
	i.textinput.SetValue(value)
}

// Validate runs the validator and returns any error
func (i *Input) Validate() error {
	if i.validator != nil {
		i.err = i.validator(i.textinput.Value())
		return i.err
	}
	return nil
}

// Error returns the current validation error
func (i Input) Error() error {
	return i.err
}

// Init implements tea.Model
func (i Input) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (i Input) Update(msg tea.Msg) (Input, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Validate on enter
			i.Validate()
			return i, nil
		case tea.KeyEsc:
			i.Blur()
			return i, nil
		}
	}

	// Update the underlying textinput
	i.textinput, cmd = i.textinput.Update(msg)
	return i, cmd
}

// View implements tea.Model
func (i Input) View() string {
	var parts []string

	labelStyle := styles.LabelStyle
	if i.focused {
		labelStyle = labelStyle.Foreground(styles.Purple).Bold(true)
	}

	// Label
	if i.label != "" {
		parts = append(parts, labelStyle.Render(i.label))
	}

	// Input box
	inputView := i.textinput.View()

	// Default style for the input box
	boxStyle := lipgloss.NewStyle().
		Padding(0, 1)

	if i.focused {
		boxStyle = boxStyle.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Purple)
	} else {
		boxStyle = boxStyle.
			Border(lipgloss.NormalBorder()).
			BorderForeground(styles.DarkGray)
	}

	// Apply width to the box style to ensure full-width look if configured
	if i.textinput.Width > 0 {
		boxStyle = boxStyle.Width(i.textinput.Width)
	}

	parts = append(parts, boxStyle.Render(inputView))

	// Error message
	if i.err != nil {
		parts = append(parts, styles.ErrorTextStyle.Render("✗ "+i.err.Error()))
	} else if i.help != "" && i.focused {
		// Only show help when focused and no error
		parts = append(parts, styles.DimStyle.Render(i.help))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// HiddenInput is an input component that masks the typed value (for passwords, API keys)
type HiddenInput struct {
	Input
}

// NewHiddenInput creates a new hidden input component
func NewHiddenInput(label, placeholder string) HiddenInput {
	input := NewInput(label, placeholder)
	input.textinput.EchoMode = textinput.EchoPassword
	input.textinput.EchoCharacter = '•'
	return HiddenInput{Input: input}
}

// Update overrides Input.Update to return HiddenInput
func (hi HiddenInput) Update(msg tea.Msg) (HiddenInput, tea.Cmd) {
	var cmd tea.Cmd
	hi.Input, cmd = hi.Input.Update(msg)
	return hi, cmd
}

// WithHelp adds help text to the hidden input
func (hi HiddenInput) WithHelp(help string) HiddenInput {
	hi.Input = hi.Input.WithHelp(help)
	return hi
}

package vendor

import (
	"encoding/json"
	"fmt"
	"openjudges/config"
	"openjudges/internal/tui/components"
	vendorpkg "openjudges/internal/vendor"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ListModel struct {
	Config  *config.Config
	List    components.SimpleSelect
	Vendors []vendorpkg.VendorConfig // Vendors loaded from files

	Width    int
	Height   int
	Selected *vendorpkg.VendorConfig
	Quitting bool
}

func NewListModel(cfg *config.Config) ListModel {
	m := ListModel{
		Config: cfg,
	}

	m.refreshList()
	return m
}

func (m *ListModel) refreshList() {
	// Load vendors from files in vendors/ folder
	// Auto-detect vendors directory location
	vendors, err := vendorpkg.LoadAllVendorsAuto()
	if err != nil {
		vendors = []vendorpkg.VendorConfig{}
	}

	m.Vendors = vendors

	var items []components.SimpleSelectItem
	for _, v := range vendors {
		// Truncate URL for display
		displayURL := v.URL
		if len(displayURL) > 35 {
			displayURL = displayURL[:32] + "..."
		}

		desc := fmt.Sprintf("%s - %s", v.Method, displayURL)
		if v.ParseAs != "" {
			desc += fmt.Sprintf(" | Parse: %s", v.ParseAs)
		}
		if len(v.Guardrail) > 0 {
			desc += fmt.Sprintf(" | GR: %d", len(v.Guardrail))
		}

		items = append(items, components.SimpleSelectItem{
			Title:       v.Name,
			Description: fmt.Sprintf("File: %s | %s", v.Filename, desc),
			Value:       v.Name,
		})
	}

	if len(items) == 0 {
		items = []components.SimpleSelectItem{
			{Title: "No vendors found", Description: "Create JSON files in vendors/ folder", Value: "none"},
		}
	}

	m.List = components.NewSimpleSelect("List Vendor", items)
	m.List.Focus()
	if m.Width > 0 {
		m.List.SetWidth(m.Width)
	}
}

func (m ListModel) Init() tea.Cmd {
	return nil
}

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Note: Delete is disabled for file-based vendors
		// Users should delete the JSON file directly from vendors/ folder
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.List.SetWidth(msg.Width)
	}

	m.List, cmd = m.List.Update(msg)

	if msgTypeIsEnter(msg) {
		val := m.List.SelectedValue()
		if val != "" && val != "none" {
			for i := range m.Vendors {
				if m.Vendors[i].Name == val {
					m.Selected = &m.Vendors[i]
					break
				}
			}
		}
	}

	return m, cmd
}

func (m ListModel) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.List.View(),
		"\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Esc: back"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true).Render("Tip: Edit/delete vendor files directly in vendors/ folder"),
	)
}

// ViewModel displays the JSON content of a vendor file
type ViewModel struct {
	Vendor      vendorpkg.VendorConfig
	JSONContent string
	Width       int
	Height      int
}

func NewViewModel(vendor vendorpkg.VendorConfig) ViewModel {
	// Convert vendor to JSON for display
	jsonBytes, _ := json.MarshalIndent(vendor, "", "  ")
	return ViewModel{
		Vendor:      vendor,
		JSONContent: string(jsonBytes),
	}
}

func (m ViewModel) Init() tea.Cmd {
	return nil
}

func (m ViewModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Any key to go back
		if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "q" {
			// Will be handled by parent model
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}

	return m, cmd
}

func (m ViewModel) View() string {
	content := lipgloss.NewStyle().
		Width(m.Width - 4).
		Render(m.JSONContent)

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Render(
				lipgloss.JoinVertical(lipgloss.Left,
					lipgloss.NewStyle().
						Foreground(lipgloss.Color("205")).
						Bold(true).
						Render(fmt.Sprintf("📄 %s (%s)", m.Vendor.Name, m.Vendor.Filename)),
					"\n",
					content,
				),
			),
		"\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Esc: back to list"),
	)
}

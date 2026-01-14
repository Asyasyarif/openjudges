package vendor

import tea "github.com/charmbracelet/bubbletea"

func msgTypeIsEnter(msg tea.Msg) bool {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		return kmsg.String() == "enter"
	}
	return false
}

func msgTypeIsTab(msg tea.Msg) bool {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		return kmsg.String() == "tab"
	}
	return false
}

func isKey(msg tea.Msg, key string) bool {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		return kmsg.String() == key
	}
	return false
}

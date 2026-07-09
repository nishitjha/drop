package internal

import "charm.land/lipgloss/v2"

type IconConfig struct {
	Positive    string
	Negative    string
	Information string
	Warning string
	Bye string
	Fact string
}

var Icons = IconConfig{
	Positive:    lipgloss.NewStyle().Foreground(lipgloss.Color("#02BA80")).Render("✔"),
	Negative:    lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5C57")).Render("✖"),
	Information: lipgloss.NewStyle().Foreground(lipgloss.Color("#3b62f1")).Render("?"),
	Warning:     lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAF00")).Render("⚠"),
	Bye:         lipgloss.NewStyle().Render("👋"),
	Fact: 	  lipgloss.NewStyle().Foreground(lipgloss.Color("#3b62f1")).Render("💡"),
}
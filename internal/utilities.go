package internal

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

type IconConfig struct {
	Positive    string
	Negative    string
	Information string
	Warning string
	Bye string
	Fact string
}

var (
	FileStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39")) 
	FolderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	TextStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("120"))
)

var Icons = IconConfig{
	Positive:    lipgloss.NewStyle().Foreground(lipgloss.Color("#02BA80")).Render("✔"),
	Negative:    lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5C57")).Render("✖"),
	Information: lipgloss.NewStyle().Foreground(lipgloss.Color("#3b62f1")).Render("?"),
	Warning:     lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAF00")).Render("⚠"),
	Bye:         lipgloss.NewStyle().Render("👋"),
	Fact: 	  lipgloss.NewStyle().Foreground(lipgloss.Color("#3b62f1")).Render("💡"),
}

func FormatBytes(b int64) string {
    if b < 1000 {
        return fmt.Sprintf("%d B", b)
    }
    div, exp := int64(1000), 0
    for n := b / 1000; n >= 1000; n /= 1000 {
        div *= 1000
        exp++
    }
    return fmt.Sprintf("%.2f %cB.", float64(b)/float64(div), "KMGTPE"[exp])
}
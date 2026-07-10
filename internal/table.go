package internal

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func PrintTable(headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = lipgloss.Width(h)
	}

	for _, row := range rows {
		for i, col := range row {
			if i < len(widths) {
				w := lipgloss.Width(col)
				if w > widths[i] {
					widths[i] = w
				}
			}
		}
	}

	var styles []lipgloss.Style
	totalWidth := 0
	for _, w := range widths {
		styles = append(styles, lipgloss.NewStyle().Width(w+4))
		totalWidth += w + 4
	}

	fmt.Println()
	for i, h := range headers {
		fmt.Print(styles[i].Bold(true).Render(h))
	}
	fmt.Println()

	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("─", totalWidth)))

	for _, row := range rows {
		for i, col := range row {
			if i < len(styles) {
				fmt.Print(styles[i].Render(col))
			}
		}
		fmt.Println()
	}
	fmt.Println()
}
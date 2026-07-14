package internal

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

const maxDescWidth = 50

func PrintTable(headers []string, rows [][]string, vertSpacing bool) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = lipgloss.Width(h)
	}

	for _, row := range rows {
		for i, col := range row {
			if i >= len(widths) {
				continue
			}
			w := lipgloss.Width(col)
			if i == len(widths)-1 && w > maxDescWidth {
				w = maxDescWidth
			}
			if w > widths[i] {
				widths[i] = w
			}
		}
	}

	var styles []lipgloss.Style
	totalWidth := 0
	for _, w := range widths {
		s := lipgloss.NewStyle().Width(w + 4)
		styles = append(styles, s)
		totalWidth += w + 4
	}

	fmt.Println()

	headerCells := make([]string, len(headers))
	for i, h := range headers {
		headerCells[i] = styles[i].Bold(true).Render(h)
	}
	fmt.Println(lipgloss.JoinHorizontal(lipgloss.Top, headerCells...))

	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("─", totalWidth)))

	for i, row := range rows {
		cells := make([]string, len(row))
		for j, col := range row {
			if j < len(styles) {
				cells[j] = styles[j].Render(col)
			}
		}
		fmt.Println(lipgloss.JoinHorizontal(lipgloss.Top, cells...))

		if vertSpacing && i < len(rows)-1 {
			fmt.Println()
		}
	}
	fmt.Println()
}
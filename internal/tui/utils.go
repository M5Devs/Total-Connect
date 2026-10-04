package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// centerOverlay centers an overlay string inside a container of width x height.
func centerOverlay(overlay string, width, height int) string {
	if width <= 0 || height <= 0 {
		return overlay
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, overlay)
}

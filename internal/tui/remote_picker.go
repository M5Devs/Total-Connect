package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type RemotePickerModel struct {
	Remotes  []string
	Selected int
	IsOpen   bool
}

func NewRemotePickerModel() RemotePickerModel {
	return RemotePickerModel{
		Remotes:  []string{},
		Selected: 0,
		IsOpen:   false,
	}
}

func (r *RemotePickerModel) Open(remotes []string) {
	// Always include "local" as the first option
	options := []string{"local"}
	for _, remote := range remotes {
		if remote != "" && remote != "local" {
			options = append(options, remote)
		}
	}

	r.Remotes = options
	r.Selected = 0
	r.IsOpen = true
}

func (r *RemotePickerModel) Close() {
	r.IsOpen = false
}

func (r *RemotePickerModel) MoveUp() {
	if r.Selected > 0 {
		r.Selected--
	}
}

func (r *RemotePickerModel) MoveDown() {
	if r.Selected < len(r.Remotes)-1 {
		r.Selected++
	}
}

func (r *RemotePickerModel) SelectedRemote() string {
	if len(r.Remotes) == 0 || r.Selected < 0 || r.Selected >= len(r.Remotes) {
		return "local"
	}
	return r.Remotes[r.Selected]
}

func (r RemotePickerModel) View(styles Styles, width, height int) string {
	if !r.IsOpen {
		return ""
	}

	var items []string
	items = append(items, styles.ModalTitle.Render("Select Storage Remote"))

	for i, rem := range r.Remotes {
		label := rem
		if rem != "local" {
			label = rem + ":"
		}

		if i == r.Selected {
			items = append(items, styles.ModalSelected.Render(fmt.Sprintf("> %s", label)))
		} else {
			items = append(items, styles.ModalItem.Render(fmt.Sprintf("  %s", label)))
		}
	}

	items = append(items, "")
	items = append(items, styles.StatusText.Render("[Enter] Select  [Esc/q] Cancel"))

	modalContent := lipgloss.JoinVertical(lipgloss.Left, items...)
	box := styles.ModalBox.Render(modalContent)

	return centerOverlay(box, width, height)
}

func centerOverlay(overlay string, width, height int) string {
	if width <= 0 || height <= 0 {
		return overlay
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, overlay)
}

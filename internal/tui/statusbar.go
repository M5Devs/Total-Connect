package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type StatusBarModel struct {
	Message string
	IsError bool
}

func NewStatusBarModel() StatusBarModel {
	return StatusBarModel{}
}

func (s *StatusBarModel) SetMessage(msg string, isError bool) {
	s.Message = msg
	s.IsError = isError
}

func (s StatusBarModel) View(styles Styles, width int) string {
	keys := []struct {
		key  string
		desc string
	}{
		{"F5", "Copy"},
		{"F6", "Move"},
		{"F7", "Mkdir"},
		{"F8", "Delete"},
		{"r", "Remote"},
		{"Tab", "Switch"},
		{"q", "Quit"},
	}

	var fnBars []string
	for _, k := range keys {
		keyStr := styles.StatusKey.Render(k.key)
		descStr := styles.StatusDesc.Render(k.desc)
		fnBars = append(fnBars, lipgloss.JoinHorizontal(lipgloss.Center, keyStr, descStr))
	}

	keyRow := strings.Join(fnBars, " ")

	msgRow := ""
	if s.Message != "" {
		if s.IsError {
			msgRow = styles.StatusError.Render("ERROR: " + s.Message)
		} else {
			msgRow = styles.StatusText.Render(s.Message)
		}
	} else {
		msgRow = styles.StatusText.Render("Ready")
	}

	full := lipgloss.JoinVertical(lipgloss.Left,
		truncateOrPad(msgRow, width),
		truncateOrPad(keyRow, width),
	)

	return styles.StatusBar.Width(width).Render(full)
}

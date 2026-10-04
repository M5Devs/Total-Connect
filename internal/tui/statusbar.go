package tui

import (
	"fmt"
	"strings"

	"github.com/M5Devs/Total-Connect/internal/models"
	"github.com/charmbracelet/lipgloss"
)

type StatusBarModel struct {
	Message  string
	IsError  bool
	Progress *models.Progress
}

func NewStatusBarModel() StatusBarModel {
	return StatusBarModel{}
}

func (s *StatusBarModel) SetMessage(msg string, isError bool) {
	s.Message = msg
	s.IsError = isError
}

func (s *StatusBarModel) SetProgress(p *models.Progress) {
	s.Progress = p
}

func (s *StatusBarModel) ClearProgress() {
	s.Progress = nil
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
		{"?", "Help"},
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
	if s.Progress != nil {
		pct := s.Progress.Percentage
		if pct > 100 {
			pct = 100
		}
		file := s.Progress.CurrentFile
		if file == "" {
			file = "file"
		}

		var progStr string
		if s.Progress.TotalBytes > 0 {
			progStr = fmt.Sprintf("Transferring %s: %.1f%% (%s / %s)", file, pct, formatSize(s.Progress.BytesTransferred), formatSize(s.Progress.TotalBytes))
		} else {
			progStr = fmt.Sprintf("Transferring %s... (%.1f%%)", file, pct)
		}
		msgRow = styles.StatusText.Render(progStr)
	} else if s.Message != "" {
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

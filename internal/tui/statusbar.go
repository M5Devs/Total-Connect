package tui

import (
	"fmt"
	"strings"

	"github.com/M5Devs/Total-Connect/internal/models"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

type StatusBarModel struct {
	Message     string
	IsError     bool
	Progress    *models.Progress
	ProgressBar progress.Model
}

func NewStatusBarModel() StatusBarModel {
	p := progress.New(progress.WithDefaultGradient())
	return StatusBarModel{
		ProgressBar: p,
	}
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
		{"F2", "Rename"},
		{"F5", "Copy"},
		{"F6", "Move"},
		{"F7", "Mkdir"},
		{"F8", "Delete"},
		{"^B", "Bkmark"},
		{"b", "Jump"},
		{"?", "Help"},
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
		pct := s.Progress.Percentage / 100.0
		if pct > 1.0 {
			pct = 1.0
		} else if pct < 0.0 {
			pct = 0.0
		}
		file := s.Progress.CurrentFile
		if file == "" {
			file = "file"
		}

		s.ProgressBar.Width = width - len(file) - 30
		if s.ProgressBar.Width < 10 {
			s.ProgressBar.Width = 10
		}

		barView := s.ProgressBar.ViewAs(pct)
		if s.Progress.TotalBytes > 0 {
			msgRow = fmt.Sprintf("Copying %s %s %s / %s", file, barView, formatSize(s.Progress.BytesTransferred), formatSize(s.Progress.TotalBytes))
		} else {
			msgRow = fmt.Sprintf("Copying %s %s", file, barView)
		}
		msgRow = styles.StatusText.Render(msgRow)
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

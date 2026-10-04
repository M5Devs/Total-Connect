package tui

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/M5Devs/Total-Connect/internal/models"
	"github.com/charmbracelet/lipgloss"
)

type PaneModel struct {
	ID        string
	Path      string
	Items     []models.FileItem
	Cursor    int
	Scroll    int
	Width     int
	Height    int
	IsActive  bool
	IsLoading bool
}

func NewPaneModel(id string, initialPath string) PaneModel {
	return PaneModel{
		ID:        id,
		Path:      initialPath,
		Items:     []models.FileItem{},
		Cursor:    0,
		Scroll:    0,
		Width:     40,
		Height:    20,
		IsActive:  false,
		IsLoading: false,
	}
}

func (p *PaneModel) SetItems(items []models.FileItem) {
	var dirs []models.FileItem
	var files []models.FileItem

	for _, item := range items {
		if item.Name == ".." {
			continue
		}
		if item.IsDir {
			dirs = append(dirs, item)
		} else {
			files = append(files, item)
		}
	}

	// Total Commander sorting: directories A-Z case-insensitive, followed by files A-Z case-insensitive
	sort.SliceStable(dirs, func(i, j int) bool {
		ni := strings.ToLower(dirs[i].Name)
		nj := strings.ToLower(dirs[j].Name)
		if ni == nj {
			return dirs[i].Name < dirs[j].Name
		}
		return ni < nj
	})

	sort.SliceStable(files, func(i, j int) bool {
		ni := strings.ToLower(files[i].Name)
		nj := strings.ToLower(files[j].Name)
		if ni == nj {
			return files[i].Name < files[j].Name
		}
		return ni < nj
	})

	var list []models.FileItem
	if canNavigateUp(p.Path) {
		list = append(list, models.FileItem{
			Name:  "..",
			IsDir: true,
		})
	}

	list = append(list, dirs...)
	list = append(list, files...)
	p.Items = list

	if p.Cursor >= len(p.Items) {
		p.Cursor = max(0, len(p.Items)-1)
	}
	p.updateScroll()
}

func (p *PaneModel) MoveCursorUp() {
	if p.Cursor > 0 {
		p.Cursor--
		p.updateScroll()
	}
}

func (p *PaneModel) MoveCursorDown() {
	if p.Cursor < len(p.Items)-1 {
		p.Cursor++
		p.updateScroll()
	}
}

func (p *PaneModel) PageUp() {
	pageSize := p.Height - 4
	if pageSize < 1 {
		pageSize = 1
	}
	p.Cursor -= pageSize
	if p.Cursor < 0 {
		p.Cursor = 0
	}
	p.updateScroll()
}

func (p *PaneModel) PageDown() {
	pageSize := p.Height - 4
	if pageSize < 1 {
		pageSize = 1
	}
	p.Cursor += pageSize
	if p.Cursor >= len(p.Items) {
		p.Cursor = max(0, len(p.Items)-1)
	}
	p.updateScroll()
}

func (p *PaneModel) SelectedItem() *models.FileItem {
	if len(p.Items) == 0 || p.Cursor < 0 || p.Cursor >= len(p.Items) {
		return nil
	}
	return &p.Items[p.Cursor]
}

func (p *PaneModel) updateScroll() {
	visibleRows := p.Height - 4 // border top/bottom, header, title
	if visibleRows < 1 {
		visibleRows = 1
	}

	if p.Cursor < p.Scroll {
		p.Scroll = p.Cursor
	} else if p.Cursor >= p.Scroll+visibleRows {
		p.Scroll = p.Cursor - visibleRows + 1
	}
}

func (p PaneModel) View(styles Styles) string {
	borderStyle := styles.InactiveBorder
	titleStyle := styles.InactiveTitle
	if p.IsActive {
		borderStyle = styles.ActiveBorder
		titleStyle = styles.ActiveTitle
	}

	titleText := fmt.Sprintf(" %s: %s ", strings.ToUpper(p.ID), truncateString(p.Path, p.Width-8))
	title := titleStyle.Render(titleText)

	contentWidth := p.Width - 2
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Calculate visible items height
	visibleRows := p.Height - 4
	if visibleRows < 1 {
		visibleRows = 1
	}

	var lines []string

	// Header row
	header := fmt.Sprintf("%-20s %10s %19s", "Name", "Size", "ModTime")
	header = truncateOrPad(header, contentWidth)
	lines = append(lines, styles.Header.Render(header))

	if p.IsLoading {
		lines = append(lines, " Loading contents...")
	} else if len(p.Items) == 0 {
		lines = append(lines, " (empty directory)")
	} else {
		end := p.Scroll + visibleRows
		if end > len(p.Items) {
			end = len(p.Items)
		}

		for i := p.Scroll; i < end; i++ {
			item := p.Items[i]
			isSel := i == p.Cursor && p.IsActive

			nameStr := item.Name
			if item.IsDir && item.Name != ".." {
				nameStr = "/" + item.Name
			}

			sizeStr := ""
			if !item.IsDir {
				sizeStr = formatSize(item.Size)
			} else {
				sizeStr = "<DIR>"
			}

			modTimeStr := ""
			if !item.ModTime.IsZero() {
				modTimeStr = item.ModTime.Format("2006-01-02 15:04")
			}

			nameWidth := contentWidth - 32
			if nameWidth < 10 {
				nameWidth = 10
			}

			rowStr := fmt.Sprintf("%-*s %10s %19s", nameWidth, truncateString(nameStr, nameWidth), sizeStr, modTimeStr)
			rowStr = truncateOrPad(rowStr, contentWidth)

			if isSel {
				lines = append(lines, styles.SelectedRow.Render(rowStr))
			} else if item.IsDir {
				lines = append(lines, styles.DirItem.Render(rowStr))
			} else {
				lines = append(lines, styles.NormalRow.Render(rowStr))
			}
		}
	}

	// Fill remaining height with empty lines if needed
	for len(lines) < visibleRows+1 {
		lines = append(lines, "")
	}

	body := lipgloss.JoinVertical(lipgloss.Left, lines...)

	box := borderStyle.Width(p.Width - 2).Height(p.Height - 2).Render(body)
	return boxOverlayTitle(box, title)
}

func canNavigateUp(currentPath string) bool {
	if currentPath == "" || currentPath == "." || currentPath == "/" {
		return false
	}
	if strings.HasSuffix(currentPath, ":") || strings.HasSuffix(currentPath, ":/") {
		return false
	}
	return true
}

func GetParentPath(currentPath string) string {
	if currentPath == "" || currentPath == "." {
		return "."
	}

	if idx := strings.Index(currentPath, ":"); idx != -1 {
		remote := currentPath[:idx+1]
		sub := currentPath[idx+1:]
		sub = strings.TrimPrefix(sub, "/")
		sub = strings.TrimSuffix(sub, "/")
		if sub == "" || sub == "." {
			return remote
		}
		parentSub := path.Dir(sub)
		if parentSub == "." || parentSub == "/" || parentSub == "" {
			return remote
		}
		return remote + parentSub
	}

	// Local path
	clean := filepath.Clean(currentPath)
	parent := filepath.Dir(clean)
	if parent == clean {
		return currentPath
	}
	return parent
}

func JoinPath(currentPath, child string) string {
	if child == ".." {
		return GetParentPath(currentPath)
	}
	if currentPath == "" || currentPath == "." {
		return child
	}
	if strings.HasSuffix(currentPath, ":") {
		return currentPath + child
	}
	if idx := strings.Index(currentPath, ":"); idx != -1 {
		return strings.TrimSuffix(currentPath, "/") + "/" + child
	}
	return filepath.Join(currentPath, child)
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func truncateString(s string, l int) string {
	if l <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= l {
		return s
	}
	if l <= 3 {
		return string(runes[:l])
	}
	return string(runes[:l-3]) + "..."
}

func truncateOrPad(s string, width int) string {
	runes := []rune(s)
	if len(runes) > width {
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func boxOverlayTitle(box string, title string) string {
	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}

	topLine := []rune(lines[0])
	titleRunes := []rune(title)

	if len(topLine) > len(titleRunes)+4 {
		// Embed title into top line starting at index 2
		copy(topLine[2:], titleRunes)
		lines[0] = string(topLine)
	}

	return strings.Join(lines, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

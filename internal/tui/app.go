package tui

import (
	"context"
	"fmt"
	"strings"

	tcconfig "github.com/M5Devs/Total-Connect/internal/config"
	"github.com/M5Devs/Total-Connect/internal/core"
	"github.com/M5Devs/Total-Connect/internal/models"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ModalType int

const (
	ModalNone ModalType = iota
	ModalRemotePicker
	ModalMkdir
	ModalConfirmDelete
	ModalHelp
	ModalRename
	ModalPatternSelect
	ModalAddBookmark
	ModalBookmarks
)

// Messages for async operations
type EntriesLoadedMsg struct {
	PaneID string
	Path   string
	Items  []models.FileItem
	Err    error
}

type OperationCompleteMsg struct {
	Op  string
	Err error
}

type RemotesLoadedMsg struct {
	Remotes []string
	Err     error
}

type ProgressMsg models.Progress

type ProgressFinishedMsg struct{}

type AppModel struct {
	engine core.StorageEngine
	styles Styles

	LeftPane  PaneModel
	RightPane PaneModel
	ActiveID  string

	StatusBar    StatusBarModel
	RemotePicker RemotePickerModel
	Spinner      spinner.Model

	ActiveModal ModalType
	TextInput   textinput.Model // For mkdir prompt

	RenameInput   textinput.Model
	PatternInput  textinput.Model
	BookmarkInput textinput.Model

	BookmarksList  []tcconfig.Bookmark
	BookmarkCursor int

	PendingRenameItem string
	PendingDeletePath string

	Width  int
	Height int
}

func NewAppModel(engine core.StorageEngine) AppModel {
	s := spinner.New()
	s.Spinner = spinner.Dot

	ti := textinput.New()
	ti.Placeholder = "New directory name..."
	ti.CharLimit = 156
	ti.Width = 30

	ri := textinput.New()
	ri.Placeholder = "New name..."
	ri.CharLimit = 156
	ri.Width = 30

	pi := textinput.New()
	pi.Placeholder = "Pattern (e.g. *.png, *.go)..."
	pi.CharLimit = 156
	pi.Width = 30

	bi := textinput.New()
	bi.Placeholder = "Bookmark name..."
	bi.CharLimit = 156
	bi.Width = 30

	left := NewPaneModel("left", ".")
	left.IsActive = true

	right := NewPaneModel("right", ".")
	right.IsActive = false

	return AppModel{
		engine:        engine,
		styles:        DefaultStyles(),
		LeftPane:      left,
		RightPane:     right,
		ActiveID:      "left",
		StatusBar:     NewStatusBarModel(),
		RemotePicker:  NewRemotePickerModel(),
		Spinner:       s,
		ActiveModal:   ModalNone,
		TextInput:     ti,
		RenameInput:   ri,
		PatternInput:  pi,
		BookmarkInput: bi,
	}
}

func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.Spinner.Tick,
		m.fetchPaneEntries("left", m.LeftPane.Path),
		m.fetchPaneEntries("right", m.RightPane.Path),
	)
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.recalculatePaneSizes()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		cmds = append(cmds, cmd)

	case ProgressMsg:
		p := models.Progress(msg)
		m.StatusBar.SetProgress(&p)

	case ProgressFinishedMsg:
		// Async progress finished, no state mutation required

	case EntriesLoadedMsg:
		if msg.PaneID == "left" {
			m.LeftPane.IsLoading = false
			if msg.Err != nil {
				m.StatusBar.SetMessage(fmt.Sprintf("Left pane error: %v", msg.Err), true)
			} else {
				m.LeftPane.Path = msg.Path
				m.LeftPane.SetItems(msg.Items)
			}
		} else if msg.PaneID == "right" {
			m.RightPane.IsLoading = false
			if msg.Err != nil {
				m.StatusBar.SetMessage(fmt.Sprintf("Right pane error: %v", msg.Err), true)
			} else {
				m.RightPane.Path = msg.Path
				m.RightPane.SetItems(msg.Items)
			}
		}

	case RemotesLoadedMsg:
		if msg.Err != nil {
			m.StatusBar.SetMessage(fmt.Sprintf("Error fetching remotes: %v", msg.Err), true)
		} else {
			m.RemotePicker.Open(msg.Remotes)
			m.ActiveModal = ModalRemotePicker
		}

	case OperationCompleteMsg:
		m.StatusBar.ClearProgress()
		if msg.Err != nil {
			m.StatusBar.SetMessage(fmt.Sprintf("%s failed: %v", msg.Op, msg.Err), true)
		} else {
			m.StatusBar.SetMessage(fmt.Sprintf("%s completed successfully", msg.Op), false)
		}
		// Clear pane selections after batch operations
		m.LeftPane.ClearSelection()
		m.RightPane.ClearSelection()

		// Refresh both panes after any file op
		m.LeftPane.IsLoading = true
		m.RightPane.IsLoading = true
		cmds = append(cmds,
			m.fetchPaneEntries("left", m.LeftPane.Path),
			m.fetchPaneEntries("right", m.RightPane.Path),
		)

	case tea.KeyMsg:
		// Handle active modal input first
		if m.ActiveModal != ModalNone {
			return m.handleModalKeyMsg(msg)
		}

		// Main navigation keybindings
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "?":
			m.ActiveModal = ModalHelp

		case "tab":
			if m.ActiveID == "left" {
				m.ActiveID = "right"
				m.LeftPane.IsActive = false
				m.RightPane.IsActive = true
			} else {
				m.ActiveID = "left"
				m.LeftPane.IsActive = true
				m.RightPane.IsActive = false
			}

		case "up", "k":
			m.getActivePane().MoveCursorUp()

		case "down", "j":
			m.getActivePane().MoveCursorDown()

		case "pgup":
			m.getActivePane().PageUp()

		case "pgdown":
			m.getActivePane().PageDown()

		case " ":
			m.getActivePane().ToggleSelection()

		case "+":
			m.PatternInput.Reset()
			m.PatternInput.Focus()
			m.ActiveModal = ModalPatternSelect

		case "f2", "e":
			active := m.getActivePane()
			item := active.SelectedItem()
			if item != nil && item.Name != ".." {
				m.PendingRenameItem = item.Name
				m.RenameInput.SetValue(item.Name)
				m.RenameInput.Focus()
				m.ActiveModal = ModalRename
			}

		case "ctrl+b":
			active := m.getActivePane()
			m.BookmarkInput.SetValue(fmt.Sprintf("%s (%s)", getRemoteFromPath(active.Path), active.Path))
			m.BookmarkInput.Focus()
			m.ActiveModal = ModalAddBookmark

		case "b":
			bms, err := tcconfig.LoadBookmarks()
			if err != nil {
				m.StatusBar.SetMessage(fmt.Sprintf("Failed to load bookmarks: %v", err), true)
			} else {
				m.BookmarksList = bms
				m.BookmarkCursor = 0
				m.ActiveModal = ModalBookmarks
			}

		case "enter":
			active := m.getActivePane()
			item := active.SelectedItem()
			if item != nil {
				if item.Name == ".." {
					newPath := GetParentPath(active.Path)
					active.IsLoading = true
					cmds = append(cmds, m.fetchPaneEntries(m.ActiveID, newPath))
				} else if item.IsDir {
					newPath := JoinPath(active.Path, item.Name)
					active.IsLoading = true
					cmds = append(cmds, m.fetchPaneEntries(m.ActiveID, newPath))
				}
			}

		case "backspace":
			active := m.getActivePane()
			if canNavigateUp(active.Path) {
				newPath := GetParentPath(active.Path)
				active.IsLoading = true
				cmds = append(cmds, m.fetchPaneEntries(m.ActiveID, newPath))
			}

		case "r":
			cmds = append(cmds, m.fetchRemotesCmd())

		case "f5", "c":
			active := m.getActivePane()
			inactive := m.getInactivePane()
			selected := active.GetSelectedItems()
			if len(selected) > 0 {
				m.StatusBar.SetMessage(fmt.Sprintf("Copying %d item(s)...", len(selected)), false)
				cmds = append(cmds, m.batchCopyCmd(active.Path, inactive.Path, selected))
			}

		case "f6", "m":
			active := m.getActivePane()
			inactive := m.getInactivePane()
			selected := active.GetSelectedItems()
			if len(selected) > 0 {
				m.StatusBar.SetMessage(fmt.Sprintf("Moving %d item(s)...", len(selected)), false)
				cmds = append(cmds, m.batchMoveCmd(active.Path, inactive.Path, selected))
			}

		case "f7", "n":
			m.TextInput.Reset()
			m.TextInput.Focus()
			m.ActiveModal = ModalMkdir

		case "f8", "d":
			active := m.getActivePane()
			selected := active.GetSelectedItems()
			if len(selected) > 0 {
				m.PendingDeletePath = fmt.Sprintf("%d item(s)", len(selected))
				m.ActiveModal = ModalConfirmDelete
			}

		case "ctrl+r":
			active := m.getActivePane()
			active.IsLoading = true
			cmds = append(cmds, m.fetchPaneEntries(m.ActiveID, active.Path))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) handleModalKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.ActiveModal {
	case ModalHelp:
		switch msg.String() {
		case "esc", "q", "?", "enter":
			m.ActiveModal = ModalNone
		}

	case ModalRemotePicker:
		switch msg.String() {
		case "esc", "q":
			m.RemotePicker.Close()
			m.ActiveModal = ModalNone
		case "up", "k":
			m.RemotePicker.MoveUp()
		case "down", "j":
			m.RemotePicker.MoveDown()
		case "enter":
			selectedRemote := m.RemotePicker.SelectedRemote()
			m.RemotePicker.Close()
			m.ActiveModal = ModalNone

			targetPath := "."
			if selectedRemote != "local" {
				targetPath = selectedRemote + ":"
			}

			active := m.getActivePane()
			active.IsLoading = true
			return m, m.fetchPaneEntries(m.ActiveID, targetPath)
		}

	case ModalMkdir:
		switch msg.String() {
		case "esc":
			m.ActiveModal = ModalNone
			m.TextInput.Blur()
		case "enter":
			dirName := m.TextInput.Value()
			m.ActiveModal = ModalNone
			m.TextInput.Blur()
			if dirName != "" {
				active := m.getActivePane()
				targetPath := JoinPath(active.Path, dirName)
				m.StatusBar.SetMessage(fmt.Sprintf("Creating directory %s...", dirName), false)
				return m, m.mkdirCmd(targetPath)
			}
		default:
			var cmd tea.Cmd
			m.TextInput, cmd = m.TextInput.Update(msg)
			return m, cmd
		}

	case ModalRename:
		switch msg.String() {
		case "esc":
			m.ActiveModal = ModalNone
			m.RenameInput.Blur()
		case "enter":
			newName := strings.TrimSpace(m.RenameInput.Value())
			m.ActiveModal = ModalNone
			m.RenameInput.Blur()
			if newName != "" && m.PendingRenameItem != "" && newName != m.PendingRenameItem {
				active := m.getActivePane()
				oldPath := JoinPath(active.Path, m.PendingRenameItem)
				newPath := JoinPath(active.Path, newName)
				remote := getRemoteFromPath(active.Path)
				m.StatusBar.SetMessage(fmt.Sprintf("Renaming %s to %s...", m.PendingRenameItem, newName), false)
				return m, m.renameCmd(remote, oldPath, newPath)
			}
		default:
			var cmd tea.Cmd
			m.RenameInput, cmd = m.RenameInput.Update(msg)
			return m, cmd
		}

	case ModalPatternSelect:
		switch msg.String() {
		case "esc":
			m.ActiveModal = ModalNone
			m.PatternInput.Blur()
		case "enter":
			pattern := strings.TrimSpace(m.PatternInput.Value())
			m.ActiveModal = ModalNone
			m.PatternInput.Blur()
			if pattern != "" {
				m.getActivePane().SelectPattern(pattern)
			}
		default:
			var cmd tea.Cmd
			m.PatternInput, cmd = m.PatternInput.Update(msg)
			return m, cmd
		}

	case ModalAddBookmark:
		switch msg.String() {
		case "esc":
			m.ActiveModal = ModalNone
			m.BookmarkInput.Blur()
		case "enter":
			name := strings.TrimSpace(m.BookmarkInput.Value())
			m.ActiveModal = ModalNone
			m.BookmarkInput.Blur()
			if name != "" {
				active := m.getActivePane()
				remote := getRemoteFromPath(active.Path)
				bm := tcconfig.Bookmark{
					Name:   name,
					Remote: remote,
					Path:   active.Path,
				}
				if err := tcconfig.AddBookmark(bm); err != nil {
					m.StatusBar.SetMessage(fmt.Sprintf("Failed to add bookmark: %v", err), true)
				} else {
					m.StatusBar.SetMessage(fmt.Sprintf("Added bookmark %q", name), false)
				}
			}
		default:
			var cmd tea.Cmd
			m.BookmarkInput, cmd = m.BookmarkInput.Update(msg)
			return m, cmd
		}

	case ModalBookmarks:
		switch msg.String() {
		case "esc", "q":
			m.ActiveModal = ModalNone
		case "up", "k":
			if m.BookmarkCursor > 0 {
				m.BookmarkCursor--
			}
		case "down", "j":
			if m.BookmarkCursor < len(m.BookmarksList)-1 {
				m.BookmarkCursor++
			}
		case "d", "delete":
			if len(m.BookmarksList) > 0 && m.BookmarkCursor >= 0 && m.BookmarkCursor < len(m.BookmarksList) {
				toDel := m.BookmarksList[m.BookmarkCursor]
				_ = tcconfig.DeleteBookmark(toDel.Name)
				m.BookmarksList, _ = tcconfig.LoadBookmarks()
				if m.BookmarkCursor >= len(m.BookmarksList) {
					m.BookmarkCursor = max(0, len(m.BookmarksList)-1)
				}
			}
		case "enter":
			if len(m.BookmarksList) > 0 && m.BookmarkCursor >= 0 && m.BookmarkCursor < len(m.BookmarksList) {
				selectedBm := m.BookmarksList[m.BookmarkCursor]
				m.ActiveModal = ModalNone

				active := m.getActivePane()
				active.IsLoading = true
				return m, m.fetchPaneEntries(m.ActiveID, selectedBm.Path)
			}
		}

	case ModalConfirmDelete:
		switch msg.String() {
		case "y", "Y", "enter":
			active := m.getActivePane()
			selected := active.GetSelectedItems()
			m.PendingDeletePath = ""
			m.ActiveModal = ModalNone
			if len(selected) > 0 {
				m.StatusBar.SetMessage(fmt.Sprintf("Deleting %d item(s)...", len(selected)), false)
				return m, m.batchDeleteCmd(active.Path, selected)
			}
		case "n", "N", "esc", "q":
			m.PendingDeletePath = ""
			m.ActiveModal = ModalNone
		}
	}

	return m, nil
}

func getRemoteFromPath(p string) string {
	if idx := strings.Index(p, ":"); idx != -1 {
		return p[:idx]
	}
	return "local"
}

func (m *AppModel) getActivePane() *PaneModel {
	if m.ActiveID == "left" {
		return &m.LeftPane
	}
	return &m.RightPane
}

func (m *AppModel) getInactivePane() *PaneModel {
	if m.ActiveID == "left" {
		return &m.RightPane
	}
	return &m.LeftPane
}

func (m *AppModel) recalculatePaneSizes() {
	paneWidth := m.Width / 2
	paneHeight := m.Height - 3 // leave room for 2-line status bar + gap

	if paneWidth < 10 {
		paneWidth = 10
	}
	if paneHeight < 5 {
		paneHeight = 5
	}

	m.LeftPane.Width = paneWidth
	m.LeftPane.Height = paneHeight

	m.RightPane.Width = m.Width - paneWidth
	m.RightPane.Height = paneHeight
}

func (m *AppModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing Total Connect TUI..."
	}

	leftView := m.LeftPane.View(m.styles)
	rightView := m.RightPane.View(m.styles)

	panesView := lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)
	statusBarView := m.StatusBar.View(m.styles, m.Width)

	mainLayout := lipgloss.JoinVertical(lipgloss.Left, panesView, statusBarView)

	// Render modal overlay if active
	if m.ActiveModal != ModalNone {
		var modalContent string
		switch m.ActiveModal {
		case ModalHelp:
			body := lipgloss.JoinVertical(lipgloss.Left,
				m.styles.ModalTitle.Render("Total Connect - Keyboard Shortcuts"),
				"",
				"  Tab         Switch active pane",
				"  Up/Down/j/k Move cursor up/down",
				"  PgUp/PgDown Page up/down",
				"  Space       Toggle item selection",
				"  +           Select items by pattern (e.g. *.png)",
				"  F2 / e      Inline rename selected item",
				"  Enter       Open directory / Enter",
				"  Backspace   Navigate to parent directory",
				"  F5 / c      Copy selected item(s) to opposite pane",
				"  F6 / m      Move selected item(s) to opposite pane",
				"  F7 / n      Create new directory (Mkdir)",
				"  F8 / d      Delete selected item(s)",
				"  Ctrl+B      Bookmark current pane remote & path",
				"  b           Open Bookmarks menu",
				"  r           Open Storage Remote Picker",
				"  Ctrl+R      Refresh current pane",
				"  ?           Toggle Help cheat sheet",
				"  q / Ctrl+C  Quit application",
				"",
				m.styles.StatusText.Render("[Esc / q / ? / Enter] Close Help"),
			)
			box := m.styles.ModalBox.Render(body)
			modalContent = centerOverlay(box, m.Width, m.Height)

		case ModalRemotePicker:
			modalContent = m.RemotePicker.View(m.styles, m.Width, m.Height)

		case ModalMkdir:
			body := lipgloss.JoinVertical(lipgloss.Left,
				m.styles.ModalTitle.Render("Create New Directory"),
				m.TextInput.View(),
				"",
				m.styles.StatusText.Render("[Enter] Submit  [Esc] Cancel"),
			)
			box := m.styles.ModalBox.Render(body)
			modalContent = centerOverlay(box, m.Width, m.Height)

		case ModalRename:
			body := lipgloss.JoinVertical(lipgloss.Left,
				m.styles.ModalTitle.Render("In-Place Rename"),
				m.RenameInput.View(),
				"",
				m.styles.StatusText.Render("[Enter] Rename  [Esc] Cancel"),
			)
			box := m.styles.ModalBox.Render(body)
			modalContent = centerOverlay(box, m.Width, m.Height)

		case ModalPatternSelect:
			body := lipgloss.JoinVertical(lipgloss.Left,
				m.styles.ModalTitle.Render("Select Files by Pattern"),
				m.PatternInput.View(),
				"",
				m.styles.StatusText.Render("[Enter] Select  [Esc] Cancel"),
			)
			box := m.styles.ModalBox.Render(body)
			modalContent = centerOverlay(box, m.Width, m.Height)

		case ModalAddBookmark:
			body := lipgloss.JoinVertical(lipgloss.Left,
				m.styles.ModalTitle.Render("Add Bookmark"),
				m.BookmarkInput.View(),
				"",
				m.styles.StatusText.Render("[Enter] Save Bookmark  [Esc] Cancel"),
			)
			box := m.styles.ModalBox.Render(body)
			modalContent = centerOverlay(box, m.Width, m.Height)

		case ModalBookmarks:
			var lines []string
			lines = append(lines, m.styles.ModalTitle.Render("⭐ Quick Bookmarks / Favorites"))
			lines = append(lines, "")

			if len(m.BookmarksList) == 0 {
				lines = append(lines, " (No saved bookmarks. Press Ctrl+B to add one)")
			} else {
				for i, bm := range m.BookmarksList {
					row := fmt.Sprintf(" %-20s -> %s ", bm.Name, bm.Path)
					if i == m.BookmarkCursor {
						lines = append(lines, m.styles.ModalSelected.Render(row))
					} else {
						lines = append(lines, m.styles.ModalItem.Render(row))
					}
				}
			}

			lines = append(lines, "")
			lines = append(lines, m.styles.StatusText.Render("[Enter] Jump  [d] Delete  [Esc] Close"))

			body := lipgloss.JoinVertical(lipgloss.Left, lines...)
			box := m.styles.ModalBox.Render(body)
			modalContent = centerOverlay(box, m.Width, m.Height)

		case ModalConfirmDelete:
			body := lipgloss.JoinVertical(lipgloss.Left,
				m.styles.ModalTitle.Render("Confirm Delete"),
				fmt.Sprintf("Are you sure you want to delete %s?", m.PendingDeletePath),
				"",
				m.styles.StatusText.Render("[Y/Enter] Yes  [N/Esc] No"),
			)
			box := m.styles.ModalBox.Render(body)
			modalContent = centerOverlay(box, m.Width, m.Height)
		}

		if modalContent != "" {
			return modalContent
		}
	}

	return mainLayout
}

// Async StorageEngine Cmds

func (m *AppModel) fetchPaneEntries(paneID, targetPath string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		items, err := m.engine.ListEntries(ctx, targetPath)
		return EntriesLoadedMsg{
			PaneID: paneID,
			Path:   targetPath,
			Items:  items,
			Err:    err,
		}
	}
}

func (m *AppModel) fetchRemotesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		remotes, err := m.engine.ListRemotes(ctx)
		return RemotesLoadedMsg{
			Remotes: remotes,
			Err:     err,
		}
	}
}

func (m *AppModel) batchCopyCmd(srcDir, dstDir string, items []models.FileItem) tea.Cmd {
	ch := make(chan tea.Msg, 20)
	go func() {
		defer close(ch)
		progressHandler := func(p models.Progress) {
			ch <- ProgressMsg(p)
		}
		ctx := core.WithProgressHandler(context.Background(), progressHandler)

		var lastErr error
		for _, item := range items {
			srcPath := JoinPath(srcDir, item.Name)
			dstPath := JoinPath(dstDir, item.Name)
			if err := m.engine.Copy(ctx, srcPath, dstPath); err != nil {
				lastErr = err
				break
			}
		}

		ch <- OperationCompleteMsg{
			Op:  fmt.Sprintf("Batch Copy (%d items)", len(items)),
			Err: lastErr,
		}
	}()
	return waitForProgress(ch)
}

func (m *AppModel) batchMoveCmd(srcDir, dstDir string, items []models.FileItem) tea.Cmd {
	ch := make(chan tea.Msg, 20)
	go func() {
		defer close(ch)
		progressHandler := func(p models.Progress) {
			ch <- ProgressMsg(p)
		}
		ctx := core.WithProgressHandler(context.Background(), progressHandler)

		var lastErr error
		for _, item := range items {
			srcPath := JoinPath(srcDir, item.Name)
			dstPath := JoinPath(dstDir, item.Name)
			if err := m.engine.Move(ctx, srcPath, dstPath); err != nil {
				lastErr = err
				break
			}
		}

		ch <- OperationCompleteMsg{
			Op:  fmt.Sprintf("Batch Move (%d items)", len(items)),
			Err: lastErr,
		}
	}()
	return waitForProgress(ch)
}

func (m *AppModel) batchDeleteCmd(dir string, items []models.FileItem) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var lastErr error
		for _, item := range items {
			itemPath := JoinPath(dir, item.Name)
			if err := m.engine.Delete(ctx, itemPath); err != nil {
				lastErr = err
				break
			}
		}
		return OperationCompleteMsg{
			Op:  fmt.Sprintf("Batch Delete (%d items)", len(items)),
			Err: lastErr,
		}
	}
}

func (m *AppModel) renameCmd(remote, oldPath, newPath string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.engine.Rename(ctx, remote, oldPath, newPath)
		return OperationCompleteMsg{
			Op:  fmt.Sprintf("Rename (%s -> %s)", oldPath, newPath),
			Err: err,
		}
	}
}

func (m *AppModel) mkdirCmd(path string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.engine.Mkdir(ctx, path)
		return OperationCompleteMsg{
			Op:  fmt.Sprintf("Mkdir (%s)", path),
			Err: err,
		}
	}
}

func waitForProgress(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return ProgressFinishedMsg{}
		}
		return msg
	}
}

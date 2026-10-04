package tui

import (
	"context"
	"fmt"

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

	Width  int
	Height int

	PendingDeletePath string
}

func NewAppModel(engine core.StorageEngine) AppModel {
	s := spinner.New()
	s.Spinner = spinner.Dot

	ti := textinput.New()
	ti.Placeholder = "New directory name..."
	ti.CharLimit = 156
	ti.Width = 30

	left := NewPaneModel("left", ".")
	left.IsActive = true

	right := NewPaneModel("right", ".")
	right.IsActive = false

	return AppModel{
		engine:       engine,
		styles:       DefaultStyles(),
		LeftPane:     left,
		RightPane:    right,
		ActiveID:     "left",
		StatusBar:    NewStatusBarModel(),
		RemotePicker: NewRemotePickerModel(),
		Spinner:      s,
		ActiveModal:  ModalNone,
		TextInput:    ti,
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.Spinner.Tick,
		m.fetchPaneEntries("left", m.LeftPane.Path),
		m.fetchPaneEntries("right", m.RightPane.Path),
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			item := active.SelectedItem()
			if item != nil && item.Name != ".." {
				src := JoinPath(active.Path, item.Name)
				dst := JoinPath(inactive.Path, item.Name)
				m.StatusBar.SetMessage(fmt.Sprintf("Copying %s to %s...", item.Name, inactive.Path), false)
				cmds = append(cmds, m.copyCmd(src, dst))
			}

		case "f6", "m":
			active := m.getActivePane()
			inactive := m.getInactivePane()
			item := active.SelectedItem()
			if item != nil && item.Name != ".." {
				src := JoinPath(active.Path, item.Name)
				dst := JoinPath(inactive.Path, item.Name)
				m.StatusBar.SetMessage(fmt.Sprintf("Moving %s to %s...", item.Name, inactive.Path), false)
				cmds = append(cmds, m.moveCmd(src, dst))
			}

		case "f7", "n":
			m.TextInput.Reset()
			m.TextInput.Focus()
			m.ActiveModal = ModalMkdir

		case "f8", "d":
			active := m.getActivePane()
			item := active.SelectedItem()
			if item != nil && item.Name != ".." {
				m.PendingDeletePath = JoinPath(active.Path, item.Name)
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

func (m AppModel) handleModalKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	case ModalConfirmDelete:
		switch msg.String() {
		case "y", "Y", "enter":
			target := m.PendingDeletePath
			m.PendingDeletePath = ""
			m.ActiveModal = ModalNone
			m.StatusBar.SetMessage(fmt.Sprintf("Deleting %s...", target), false)
			return m, m.deleteCmd(target)
		case "n", "N", "esc", "q":
			m.PendingDeletePath = ""
			m.ActiveModal = ModalNone
		}
	}

	return m, nil
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

func (m AppModel) View() string {
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
				"  Enter       Open directory / Enter",
				"  Backspace   Navigate to parent directory",
				"  F5 / c      Copy selected item to opposite pane",
				"  F6 / m      Move selected item to opposite pane",
				"  F7 / n      Create new directory (Mkdir)",
				"  F8 / d      Delete selected item",
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
		case ModalConfirmDelete:
			body := lipgloss.JoinVertical(lipgloss.Left,
				m.styles.ModalTitle.Render("Confirm Delete"),
				fmt.Sprintf("Are you sure you want to delete?\n%s", m.PendingDeletePath),
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

func (m AppModel) fetchPaneEntries(paneID, targetPath string) tea.Cmd {
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

func (m AppModel) fetchRemotesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		remotes, err := m.engine.ListRemotes(ctx)
		return RemotesLoadedMsg{
			Remotes: remotes,
			Err:     err,
		}
	}
}

func (m AppModel) copyCmd(src, dst string) tea.Cmd {
	ch := make(chan tea.Msg, 10)
	go func() {
		defer close(ch)
		progressHandler := func(p models.Progress) {
			ch <- ProgressMsg(p)
		}
		ctx := core.WithProgressHandler(context.Background(), progressHandler)
		err := m.engine.Copy(ctx, src, dst)
		ch <- OperationCompleteMsg{
			Op:  fmt.Sprintf("Copy (%s -> %s)", src, dst),
			Err: err,
		}
	}()
	return waitForProgress(ch)
}

func (m AppModel) moveCmd(src, dst string) tea.Cmd {
	ch := make(chan tea.Msg, 10)
	go func() {
		defer close(ch)
		progressHandler := func(p models.Progress) {
			ch <- ProgressMsg(p)
		}
		ctx := core.WithProgressHandler(context.Background(), progressHandler)
		err := m.engine.Move(ctx, src, dst)
		ch <- OperationCompleteMsg{
			Op:  fmt.Sprintf("Move (%s -> %s)", src, dst),
			Err: err,
		}
	}()
	return waitForProgress(ch)
}

func (m AppModel) mkdirCmd(path string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.engine.Mkdir(ctx, path)
		return OperationCompleteMsg{
			Op:  fmt.Sprintf("Mkdir (%s)", path),
			Err: err,
		}
	}
}

func (m AppModel) deleteCmd(path string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.engine.Delete(ctx, path)
		return OperationCompleteMsg{
			Op:  fmt.Sprintf("Delete (%s)", path),
			Err: err,
		}
	}
}

func waitForProgress(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

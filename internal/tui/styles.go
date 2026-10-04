package tui

import "github.com/charmbracelet/lipgloss"

type Styles struct {
	ActiveBorder   lipgloss.Style
	InactiveBorder lipgloss.Style
	Title          lipgloss.Style
	ActiveTitle    lipgloss.Style
	InactiveTitle  lipgloss.Style
	Header         lipgloss.Style
	SelectedRow    lipgloss.Style
	NormalRow      lipgloss.Style
	DirItem        lipgloss.Style
	FileItem       lipgloss.Style
	StatusBar      lipgloss.Style
	StatusKey      lipgloss.Style
	StatusDesc     lipgloss.Style
	StatusText     lipgloss.Style
	StatusError    lipgloss.Style
	ModalBox       lipgloss.Style
	ModalTitle     lipgloss.Style
	ModalItem      lipgloss.Style
	ModalSelected  lipgloss.Style
}

func DefaultStyles() Styles {
	s := Styles{}

	s.ActiveBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")) // Blue/Purple accent

	s.InactiveBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")) // Dark gray

	s.Title = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)

	s.ActiveTitle = s.Title.Copy().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("63"))

	s.InactiveTitle = s.Title.Copy().
		Foreground(lipgloss.Color("250")).
		Background(lipgloss.Color("238"))

	s.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	s.SelectedRow = lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)

	s.NormalRow = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	s.DirItem = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	s.FileItem = lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	s.StatusBar = lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252"))

	s.StatusKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("31")).
		Padding(0, 1)

	s.StatusDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Background(lipgloss.Color("237")).
		Padding(0, 1)

	s.StatusText = lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Padding(0, 1)

	s.StatusError = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Padding(0, 1)

	s.ModalBox = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("208")).
		Padding(1, 2).
		Background(lipgloss.Color("235"))

	s.ModalTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("208")).
		MarginBottom(1)

	s.ModalItem = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	s.ModalSelected = lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("208")).
		Bold(true).
		Padding(0, 1)

	return s
}

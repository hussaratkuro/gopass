package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hussaratkuro/gopass/internal/theme"
)

// Screen identifies which top-level view is active.
type Screen int

const (
	ScreenMenu Screen = iota
	ScreenGenerator
	ScreenManagerUnlock
	ScreenManagerFirefoxPassword
	ScreenManagerList
	ScreenManagerDetail
	ScreenManagerImporting
	ScreenManagerAddEntry
)

var menuItems = []string{"Password Generator", "Password Manager"}

// Model is the root Bubble Tea model. It owns which screen is active and
// delegates to screen-specific state held in its own fields (mirroring
// svn-tui's single-model-with-screen-enum shape).
type Model struct {
	screen    Screen
	menuIndex int

	generator generatorState
	manager   managerState

	width, height int
	quitting      bool
}

// New builds the initial application model, showing the action selector.
func New() Model {
	return Model{
		screen:    ScreenMenu,
		generator: newGeneratorState(),
		manager:   newManagerState(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, theme.Watch())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case theme.ChangedMsg:
		applyTheme(msg.Palette)
		return m, theme.Watch()

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	default:
		// Route async results (import progress, etc.) to the active screen.
		if m.screen != ScreenMenu && m.screen != ScreenGenerator {
			return m.updateManager(msg)
		}
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}

	switch m.screen {
	case ScreenMenu:
		return m.handleMenuKey(msg)
	case ScreenGenerator:
		if msg.String() == "esc" {
			m.screen = ScreenMenu
			return m, nil
		}
		return m.updateGeneratorKey(msg)
	default:
		return m.handleManagerKey(msg)
	}
}

func (m Model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quitting = true
		return m, tea.Quit
	case "up", "k":
		m.menuIndex = (m.menuIndex - 1 + len(menuItems)) % len(menuItems)
	case "down", "j":
		m.menuIndex = (m.menuIndex + 1) % len(menuItems)
	case "enter":
		switch m.menuIndex {
		case 0:
			m.screen = ScreenGenerator
		case 1:
			return m.enterManager()
		}
	}
	return m, nil
}

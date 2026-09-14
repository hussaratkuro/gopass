package ui

import (
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

type commandAction uint8

const (
	commandBack commandAction = iota
	commandQuit
	commandOpenGenerator
	commandOpenManager
	commandGeneratePassword
	commandAddCredential
	commandImportFirefox
	commandRevealPassword
	commandCopyPassword
	commandCopyUsername
	commandCopyTOTP
	commandEditCredential
	commandDeleteCredential
)

type commandItem struct {
	label  string
	action commandAction
}

type commandPalette struct {
	open   bool
	query  string
	cursor int
	items  []commandItem
}

func (m Model) commandPaletteAvailable() bool {
	switch m.screen {
	case ScreenMenu, ScreenGenerator, ScreenManagerList, ScreenManagerDetail:
		return true
	default:
		return false
	}
}

func (m *Model) openCommandPalette() {
	items := []commandItem{{label: "Quit gopass", action: commandQuit}}
	switch m.screen {
	case ScreenMenu:
		items = append([]commandItem{
			{label: "Open password generator", action: commandOpenGenerator},
			{label: "Open password manager", action: commandOpenManager},
		}, items...)
	case ScreenGenerator:
		items = append([]commandItem{
			{label: "Generate password", action: commandGeneratePassword},
			{label: "Back to main menu", action: commandBack},
		}, items...)
	case ScreenManagerList:
		items = append([]commandItem{
			{label: "Add credential", action: commandAddCredential},
			{label: "Import credentials from Firefox", action: commandImportFirefox},
			{label: "Back to main menu", action: commandBack},
		}, items...)
	case ScreenManagerDetail:
		items = append([]commandItem{
			{label: "Edit credential", action: commandEditCredential},
			{label: "Copy password", action: commandCopyPassword},
			{label: "Copy username", action: commandCopyUsername},
			{label: "Reveal or hide password", action: commandRevealPassword},
			{label: "Delete credential", action: commandDeleteCredential},
			{label: "Back to credential list", action: commandBack},
		}, items...)
		if m.manager.selected.TOTPSecret != "" {
			items = append([]commandItem{{label: "Copy current TOTP code", action: commandCopyTOTP}}, items...)
		}
	}
	m.commands = commandPalette{open: true, items: items}
}

func (m Model) updateCommandPalette(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := m.filteredCommands()
	switch key.String() {
	case "esc", "ctrl+shift+p", "ctrl+p":
		m.commands = commandPalette{}
	case "up", "ctrl+k":
		m.commands.cursor = max(0, m.commands.cursor-1)
	case "down", "ctrl+j":
		m.commands.cursor = min(max(0, len(items)-1), m.commands.cursor+1)
	case "backspace":
		query := []rune(m.commands.query)
		if len(query) > 0 {
			m.commands.query = string(query[:len(query)-1])
			m.commands.cursor = 0
		}
	case "enter":
		if len(items) == 0 {
			return m, nil
		}
		action := items[min(m.commands.cursor, len(items)-1)].action
		m.commands = commandPalette{}
		return m.executeCommand(action)
	default:
		if key.Type == tea.KeyRunes && !key.Alt {
			for _, character := range key.Runes {
				if unicode.IsPrint(character) {
					m.commands.query += string(character)
				}
			}
			m.commands.cursor = 0
		}
	}
	return m, nil
}

func (m Model) executeCommand(action commandAction) (tea.Model, tea.Cmd) {
	switch action {
	case commandOpenGenerator:
		m.screen = ScreenGenerator
		return m, nil
	case commandOpenManager:
		return m.enterManager()
	case commandGeneratePassword:
		m.generator.generate()
		return m, nil
	case commandBack:
		return m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	case commandQuit:
		return m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	case commandAddCredential:
		return m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlN})
	case commandImportFirefox:
		return m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlR})
	case commandRevealPassword:
		return m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	case commandCopyPassword:
		return m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	case commandCopyUsername:
		return m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	case commandCopyTOTP:
		return m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	case commandEditCredential:
		return m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	case commandDeleteCredential:
		return m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	}
	return m, nil
}

func (m Model) filteredCommands() []commandItem {
	var filtered []commandItem
	for _, item := range m.commands.items {
		if fuzzyCommandMatch(item.label, m.commands.query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func fuzzyCommandMatch(label, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	label = strings.ToLower(label)
	position := 0
	for _, character := range query {
		found := strings.IndexRune(label[position:], character)
		if found < 0 {
			return false
		}
		position += found + 1
	}
	return true
}

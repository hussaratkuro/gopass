package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommandPaletteIsFuzzyAndContextAware(t *testing.T) {
	model := New()
	model.openCommandPalette()
	model.commands.query = "ps mgr"
	items := model.filteredCommands()
	if len(items) != 1 || items[0].action != commandOpenManager {
		t.Fatalf("menu command match = %#v", items)
	}
	updated, _ := model.updateCommandPalette(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.(Model).commands.open {
		t.Fatal("Esc did not close command palette")
	}
}

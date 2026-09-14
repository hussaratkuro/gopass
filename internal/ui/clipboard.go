package ui

import (
	"fmt"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

const clipboardClearDelay = 30 * time.Second

type clipboardClearMsg struct {
	value   string
	version uint64
}

func (m Model) copySecret(value, label string) (tea.Model, tea.Cmd) {
	if value == "" {
		m.manager.err = fmt.Errorf("%s is empty", label)
		return m, nil
	}
	if err := clipboard.WriteAll(value); err != nil {
		m.manager.err = fmt.Errorf("copy %s: %w", label, err)
		return m, nil
	}
	m.manager.err = nil
	m.manager.status = label + " copied; clipboard clears in 30 seconds"
	m.clipboardVersion++
	version := m.clipboardVersion
	return m, tea.Tick(clipboardClearDelay, func(time.Time) tea.Msg {
		return clipboardClearMsg{value: value, version: version}
	})
}

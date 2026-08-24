// Package ui hosts the top-level application model that switches between
// the password generator and password manager screens.
package ui

import "github.com/charmbracelet/lipgloss"

// Shared color palette (Catppuccin Mocha-ish) used across all screens.
var (
	ColorTitle    = lipgloss.Color("#cba6f7")
	ColorFocused  = lipgloss.Color("#f5e0dc")
	ColorNormal   = lipgloss.Color("#6c7086")
	ColorMuted    = lipgloss.Color("#7f849c")
	ColorSelected = lipgloss.Color("#cba6f7")
	ColorError    = lipgloss.Color("#f38ba8")
	ColorSuccess  = lipgloss.Color("#a6e3a1")
)

var (
	TitleStyle    = lipgloss.NewStyle().Foreground(ColorTitle).Bold(true)
	FocusedStyle  = lipgloss.NewStyle().Foreground(ColorFocused)
	NormalStyle   = lipgloss.NewStyle().Foreground(ColorNormal)
	MutedStyle    = lipgloss.NewStyle().Foreground(ColorMuted)
	SelectedStyle = lipgloss.NewStyle().Foreground(ColorSelected)
	ErrorStyle    = lipgloss.NewStyle().Foreground(ColorError)
	SuccessStyle  = lipgloss.NewStyle().Foreground(ColorSuccess)
	HelpStyle     = lipgloss.NewStyle().Foreground(ColorNormal)
)

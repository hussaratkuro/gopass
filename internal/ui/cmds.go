package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/hussaratkuro/gopass/internal/firefox"
	"github.com/hussaratkuro/gopass/internal/vault"
)

type vaultUnlockedMsg struct {
	v        *vault.Vault
	password string
}

type vaultErrorMsg struct {
	err error
}

type importDoneMsg struct {
	entries []vault.Entry
	skipped int
	err     error
}

func unlockVaultCmd(password string) tea.Cmd {
	return func() tea.Msg {
		v, err := vault.Load(password)
		if err != nil {
			return vaultErrorMsg{err: err}
		}
		return vaultUnlockedMsg{v: v, password: password}
	}
}

func createVaultCmd(password string) tea.Cmd {
	return func() tea.Msg {
		v := vault.New()
		if err := v.Save(password); err != nil {
			return vaultErrorMsg{err: err}
		}
		return vaultUnlockedMsg{v: v, password: password}
	}
}

func importFirefoxCmd(firefoxMasterPassword string) tea.Cmd {
	return func() tea.Msg {
		profile, err := firefox.Default()
		if err != nil {
			return importDoneMsg{err: err}
		}
		result, err := firefox.Import(profile.Path, firefoxMasterPassword)
		if err != nil {
			return importDoneMsg{err: err}
		}

		entries := make([]vault.Entry, 0, len(result.Credentials))
		now := time.Now()
		for _, c := range result.Credentials {
			entries = append(entries, vault.Entry{
				ID:        "firefox|" + c.Hostname + "|" + c.Username,
				Title:     c.DisplayName(),
				URL:       c.Hostname,
				Username:  c.Username,
				Password:  c.Password,
				Source:    "firefox",
				UpdatedAt: now,
			})
		}
		return importDoneMsg{entries: entries, skipped: result.Skipped}
	}
}

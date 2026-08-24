package ui

import (
	"fmt"
	"strings"
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	switch m.screen {
	case ScreenMenu:
		return viewMenu(m.menuIndex)
	case ScreenGenerator:
		return viewGenerator(m.generator)
	case ScreenManagerUnlock:
		return viewUnlock(m.manager)
	case ScreenManagerFirefoxPassword:
		return viewFirefoxPassword(m.manager)
	case ScreenManagerList:
		return viewList(m.manager)
	case ScreenManagerDetail:
		return viewDetail(m.manager)
	case ScreenManagerImporting:
		return TitleStyle.Render("Password Manager") + "\n\n" + "Importing saved logins from Firefox...\n"
	case ScreenManagerAddEntry:
		return viewAddEntry(m.manager)
	}
	return ""
}

func viewMenu(index int) string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("gopass") + "\n\n")
	for i, item := range menuItems {
		if i == index {
			b.WriteString(FocusedStyle.Render("> "+item) + "\n")
		} else {
			b.WriteString(NormalStyle.Render("  "+item) + "\n")
		}
	}
	b.WriteString("\n" + HelpStyle.Render("↑/↓ navigate · enter select · q quit"))
	return b.String()
}

func viewUnlock(mgr managerState) string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Password Manager") + "\n\n")

	if mgr.creatingVault {
		b.WriteString("No vault found yet. Choose a password to encrypt it with.\n\n")
		if !mgr.confirmStep {
			b.WriteString(mgr.passwordInput.View() + "\n")
		} else {
			b.WriteString(MutedStyle.Render(strings.Repeat("•", len(mgr.passwordInput.Value()))) + "\n")
			b.WriteString(mgr.confirmInput.View() + "\n")
		}
	} else {
		b.WriteString("Enter your vault password to unlock it.\n\n")
		b.WriteString(mgr.passwordInput.View() + "\n")
	}

	if mgr.err != nil {
		b.WriteString("\n" + ErrorStyle.Render("Error: "+mgr.err.Error()) + "\n")
	}

	b.WriteString("\n" + HelpStyle.Render("enter confirm · esc back"))
	return b.String()
}

func viewFirefoxPassword(mgr managerState) string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Password Manager") + "\n\n")
	b.WriteString("This Firefox profile is protected by a master password.\n\n")
	b.WriteString(mgr.firefoxInput.View() + "\n")
	b.WriteString("\n" + HelpStyle.Render("enter continue · esc cancel"))
	return b.String()
}

func viewList(mgr managerState) string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Password Manager") + "\n\n")
	b.WriteString(mgr.searchInput.View() + "\n\n")

	total := len(mgr.results)
	if total == 0 {
		b.WriteString(MutedStyle.Render("No entries yet. Press ctrl+r to import from Firefox.") + "\n")
	} else {
		start, end := visibleWindow(mgr.cursor, total, maxVisibleRows)
		for i := start; i < end; i++ {
			e := mgr.results[i]
			line := fmt.Sprintf("%-32s %s", truncate(e.Title, 32), MutedStyle.Render(e.Username))
			if i == mgr.cursor {
				b.WriteString(FocusedStyle.Render("> "+line) + "\n")
			} else {
				b.WriteString(NormalStyle.Render("  "+line) + "\n")
			}
		}
		b.WriteString("\n" + MutedStyle.Render(fmt.Sprintf("showing %d-%d of %d", start+1, end, total)) + "\n")
	}

	if mgr.status != "" {
		b.WriteString("\n" + SuccessStyle.Render(mgr.status) + "\n")
	}
	if mgr.err != nil {
		b.WriteString("\n" + ErrorStyle.Render("Error: "+mgr.err.Error()) + "\n")
	}

	b.WriteString("\n" + HelpStyle.Render("type to search · ↑/↓ select · enter view · ctrl+n add · ctrl+r import from Firefox · esc back"))
	return b.String()
}

func viewDetail(mgr managerState) string {
	e := mgr.selected
	var b strings.Builder
	b.WriteString(TitleStyle.Render(e.Title) + "\n\n")
	b.WriteString(MutedStyle.Render("URL:      ") + e.URL + "\n")
	b.WriteString(MutedStyle.Render("Username: ") + e.Username + "\n")

	pw := strings.Repeat("•", len(e.Password))
	if mgr.reveal {
		pw = e.Password
	}
	b.WriteString(MutedStyle.Render("Password: ") + SelectedStyle.Render(pw) + "\n")

	if mgr.status != "" {
		b.WriteString("\n" + SuccessStyle.Render(mgr.status) + "\n")
	}
	if mgr.confirmDelete {
		b.WriteString("\n" + ErrorStyle.Render("Press 'd' again to delete this entry, any other key to cancel.") + "\n")
	}

	b.WriteString("\n" + HelpStyle.Render("r reveal · c copy password · u copy username · d delete · esc back"))
	return b.String()
}

func viewAddEntry(mgr managerState) string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Add Entry") + "\n\n")

	field := func(label string, ti string, focused bool) string {
		if focused {
			return FocusedStyle.Render("> "+label) + ti + "\n"
		}
		return NormalStyle.Render("  "+label) + ti + "\n"
	}

	b.WriteString(field("Title:    ", mgr.entryTitleInput.View(), mgr.entryFocus == entryFocusTitle))
	b.WriteString(field("URL:      ", mgr.entryURLInput.View(), mgr.entryFocus == entryFocusURL))
	b.WriteString(field("Username: ", mgr.entryUsernameInput.View(), mgr.entryFocus == entryFocusUsername))
	b.WriteString(field("Password: ", mgr.entryPasswordInput.View(), mgr.entryFocus == entryFocusPassword))

	b.WriteString("\n")
	if mgr.entryFocus == entryFocusSave {
		b.WriteString(FocusedStyle.Render("> Save") + "\n")
	} else {
		b.WriteString(NormalStyle.Render("  Save") + "\n")
	}

	if mgr.err != nil {
		b.WriteString("\n" + ErrorStyle.Render("Error: "+mgr.err.Error()) + "\n")
	}

	b.WriteString("\n" + HelpStyle.Render("↑/↓ navigate · enter next/save · esc cancel"))
	return b.String()
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

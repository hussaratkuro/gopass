package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/hussaratkuro/gopass/internal/vault"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "ctrl+n":
		return tea.KeyMsg{Type: tea.KeyCtrlN}
	case "ctrl+r":
		return tea.KeyMsg{Type: tea.KeyCtrlR}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func entryFixture(title, url, username string) vault.Entry {
	return vault.Entry{ID: "test|" + url + "|" + username, Title: title, URL: url, Username: username, Password: "x", Source: "manual"}
}

func TestGeneratorFlow(t *testing.T) {
	m := New()
	var tm tea.Model = m

	tm, _ = tm.Update(key("enter")) // select generator
	m = tm.(Model)
	if m.screen != ScreenGenerator {
		t.Fatalf("expected generator screen, got %v", m.screen)
	}

	tm, _ = tm.Update(key("1"))
	tm, _ = tm.Update(key("0"))
	m = tm.(Model)
	if m.generator.lengthInput.Value() != "10" {
		t.Fatalf("expected length input value '10', got %q", m.generator.lengthInput.Value())
	}

	// down to Upper, toggle, down to Lower, toggle, down x2 to Numbers, toggle, down to Generate, enter.
	for _, k := range []string{"down", " ", "down", " ", "down", "down", " ", "down", "enter"} {
		tm, _ = tm.Update(key(k))
	}
	m = tm.(Model)
	if m.generator.err != nil {
		t.Fatalf("unexpected generation error: %v", m.generator.err)
	}
	if len(m.generator.password) != 10 {
		t.Fatalf("expected a 10-char password, got %q (len %d)", m.generator.password, len(m.generator.password))
	}

	tm, _ = tm.Update(key("esc"))
	m = tm.(Model)
	if m.screen != ScreenMenu {
		t.Fatalf("expected esc to return to menu, got screen=%v", m.screen)
	}
}

func TestManagerVaultAndSearch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m := New()
	var tm tea.Model = m

	tm, _ = tm.Update(key("down")) // menu index -> Password Manager
	tm, _ = tm.Update(key("enter"))
	m = tm.(Model)
	if m.screen != ScreenManagerUnlock || !m.manager.creatingVault {
		t.Fatalf("expected fresh-vault unlock screen, got screen=%v creatingVault=%v", m.screen, m.manager.creatingVault)
	}

	for _, r := range "hunter2" {
		tm, _ = tm.Update(key(string(r)))
	}
	tm, _ = tm.Update(key("enter")) // move to confirm step
	m = tm.(Model)
	if !m.manager.confirmStep {
		t.Fatalf("expected confirmStep=true after first password entry")
	}
	for _, r := range "hunter2" {
		tm, _ = tm.Update(key(string(r)))
	}
	tm, cmd := tm.Update(key("enter")) // submit -> createVaultCmd
	if cmd == nil {
		t.Fatalf("expected createVaultCmd to be returned")
	}
	tm, _ = tm.Update(cmd())
	m = tm.(Model)
	if m.screen != ScreenManagerList || m.manager.vault == nil {
		t.Fatalf("expected vault created and List screen, got screen=%v err=%v", m.screen, m.manager.err)
	}

	// Seed two entries directly and check search matches by name and by URL.
	m.manager.vault.Upsert(entryFixture("github.com", "https://github.com", "alice"))
	m.manager.vault.Upsert(entryFixture("example.org", "https://example.org", "bob"))
	tm = m
	for _, r := range "git" {
		tm, _ = tm.Update(key(string(r)))
	}
	m = tm.(Model)
	if len(m.manager.results) != 1 || m.manager.results[0].Title != "github.com" {
		t.Fatalf("expected exactly github.com for query 'git', got %+v", m.manager.results)
	}

	for range "git" {
		tm, _ = tm.Update(key("backspace"))
	}
	for _, r := range "example.org" {
		tm, _ = tm.Update(key(string(r)))
	}
	m = tm.(Model)
	if len(m.manager.results) != 1 || m.manager.results[0].Username != "bob" {
		t.Fatalf("expected exactly bob for query 'example.org', got %+v", m.manager.results)
	}

	tm, _ = tm.Update(key("enter")) // open detail for the single filtered result
	m = tm.(Model)
	if m.screen != ScreenManagerDetail || m.manager.reveal {
		t.Fatalf("expected detail screen with reveal=false, got screen=%v reveal=%v", m.screen, m.manager.reveal)
	}
	tm, _ = tm.Update(key("r"))
	m = tm.(Model)
	if !m.manager.reveal {
		t.Fatalf("expected reveal=true after 'r'")
	}
	tm, _ = tm.Update(key("esc"))
	m = tm.(Model)
	if m.screen != ScreenManagerList {
		t.Fatalf("expected esc from detail to return to list, got %v", m.screen)
	}
}

func TestVisibleWindow(t *testing.T) {
	cases := []struct {
		cursor, total, maxRows int
		wantStart, wantEnd     int
	}{
		{cursor: 0, total: 5, maxRows: 15, wantStart: 0, wantEnd: 5},
		{cursor: 0, total: 100, maxRows: 15, wantStart: 0, wantEnd: 15},
		{cursor: 99, total: 100, maxRows: 15, wantStart: 85, wantEnd: 100},
		{cursor: 50, total: 100, maxRows: 15, wantStart: 43, wantEnd: 58},
	}
	for _, c := range cases {
		start, end := visibleWindow(c.cursor, c.total, c.maxRows)
		if start != c.wantStart || end != c.wantEnd {
			t.Errorf("visibleWindow(%d,%d,%d) = (%d,%d), want (%d,%d)",
				c.cursor, c.total, c.maxRows, start, end, c.wantStart, c.wantEnd)
		}
		if c.cursor < start || c.cursor >= end {
			t.Errorf("cursor %d not within window [%d,%d)", c.cursor, start, end)
		}
	}
}

func TestManagerListNavigation(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m := New()
	m.manager.vault = vault.New()
	m.manager.vaultPassword = "hunter2"
	for i := 0; i < 20; i++ {
		title := string(rune('a' + i))
		m.manager.vault.Upsert(entryFixture(title, "https://"+title+".example", "user"))
	}
	m.screen = ScreenManagerList
	m.manager.refreshResults()
	if len(m.manager.results) != 20 {
		t.Fatalf("expected 20 seeded entries, got %d", len(m.manager.results))
	}

	var tm tea.Model = m
	if m.manager.cursor != 0 {
		t.Fatalf("expected initial cursor 0, got %d", m.manager.cursor)
	}

	tm, _ = tm.Update(key("up")) // clamped at 0
	m = tm.(Model)
	if m.manager.cursor != 0 {
		t.Fatalf("expected cursor to stay at 0 on up from top, got %d", m.manager.cursor)
	}

	for i := 0; i < 19; i++ {
		tm, _ = tm.Update(key("down"))
	}
	m = tm.(Model)
	if m.manager.cursor != 19 {
		t.Fatalf("expected cursor at last index 19, got %d", m.manager.cursor)
	}

	tm, _ = tm.Update(key("down")) // clamped at last index
	m = tm.(Model)
	if m.manager.cursor != 19 {
		t.Fatalf("expected cursor to stay at 19 on down from bottom, got %d", m.manager.cursor)
	}
}

func TestManagerAddAndDeleteEntry(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m := New()
	m.manager.vault = vault.New()
	m.manager.vaultPassword = "hunter2"
	m.screen = ScreenManagerList
	m.manager.refreshResults()

	var tm tea.Model = m
	tm, _ = tm.Update(key("ctrl+n"))
	m = tm.(Model)
	if m.screen != ScreenManagerAddEntry {
		t.Fatalf("expected add-entry screen, got %v", m.screen)
	}

	for _, r := range "My Site" {
		tm, _ = tm.Update(key(string(r)))
	}
	tm, _ = tm.Update(key("down")) // -> URL
	for _, r := range "https://my.site" {
		tm, _ = tm.Update(key(string(r)))
	}
	tm, _ = tm.Update(key("down")) // -> Username
	for _, r := range "carol" {
		tm, _ = tm.Update(key(string(r)))
	}
	tm, _ = tm.Update(key("down")) // -> Password
	for _, r := range "s3cret" {
		tm, _ = tm.Update(key(string(r)))
	}
	tm, _ = tm.Update(key("down")) // -> TOTP
	tm, _ = tm.Update(key("down")) // -> Save
	m = tm.(Model)
	if m.manager.entryFocus != entryFocusSave {
		t.Fatalf("expected focus on Save, got %d", m.manager.entryFocus)
	}
	tm, _ = tm.Update(key("enter")) // submit
	m = tm.(Model)
	if m.screen != ScreenManagerList {
		t.Fatalf("expected to return to list after save, got %v err=%v", m.screen, m.manager.err)
	}
	if len(m.manager.vault.Entries) != 1 {
		t.Fatalf("expected 1 entry in vault, got %d", len(m.manager.vault.Entries))
	}
	added := m.manager.vault.Entries[0]
	if added.Title != "My Site" || added.URL != "https://my.site" || added.Username != "carol" || added.Password != "s3cret" {
		t.Fatalf("unexpected saved entry: %+v", added)
	}

	// Delete requires pressing 'd' twice.
	tm, _ = tm.Update(key("enter")) // open detail
	m = tm.(Model)
	if m.screen != ScreenManagerDetail {
		t.Fatalf("expected detail screen, got %v", m.screen)
	}
	tm, _ = tm.Update(key("d"))
	m = tm.(Model)
	if !m.manager.confirmDelete || len(m.manager.vault.Entries) != 1 {
		t.Fatalf("expected delete to require confirmation first")
	}
	tm, _ = tm.Update(key("d"))
	m = tm.(Model)
	if m.screen != ScreenManagerList || len(m.manager.vault.Entries) != 0 {
		t.Fatalf("expected entry deleted and back at list, got screen=%v entries=%d", m.screen, len(m.manager.vault.Entries))
	}
}

func TestManagerEditsEntryWithoutChangingItsIdentity(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New()
	m.manager.vault = vault.New()
	m.manager.vaultPassword = "hunter2"
	entry := entryFixture("Old title", "https://example.test", "alice")
	entry.TOTPSecret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	m.manager.vault.Upsert(entry)
	m.manager.refreshResults()
	m.manager.selected = entry
	m.screen = ScreenManagerDetail

	var tm tea.Model = m
	tm, _ = tm.Update(key("e"))
	m = tm.(Model)
	if m.screen != ScreenManagerAddEntry || !m.manager.entryEditing || m.manager.entryTOTPInput.Value() != entry.TOTPSecret {
		t.Fatalf("edit form not populated: screen=%v editing=%t", m.screen, m.manager.entryEditing)
	}
	m.manager.entryTitleInput.SetValue("New title")
	m.manager.entryFocus = entryFocusSave
	tm = m
	tm, _ = tm.Update(key("enter"))
	m = tm.(Model)
	if len(m.manager.vault.Entries) != 1 || m.manager.vault.Entries[0].ID != entry.ID || m.manager.vault.Entries[0].Title != "New title" {
		t.Fatalf("edit changed identity or duplicated entry: %#v", m.manager.vault.Entries)
	}
}

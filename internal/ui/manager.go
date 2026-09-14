package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hussaratkuro/gopass/internal/firefox"
	"github.com/hussaratkuro/gopass/internal/totp"
	"github.com/hussaratkuro/gopass/internal/vault"
)

// maxVisibleRows caps how many entries are drawn at once in the list screen;
// the window scrolls to keep the cursor in view.
const maxVisibleRows = 15

const (
	entryFocusTitle = iota
	entryFocusURL
	entryFocusUsername
	entryFocusPassword
	entryFocusTOTP
	entryFocusSave
	entryFocusCount
)

// managerState holds all state for the password manager screens. The vault,
// once unlocked, and its password are kept in memory for the rest of the
// session so switching back to the menu and returning doesn't re-prompt.
type managerState struct {
	vault         *vault.Vault
	vaultPassword string
	creatingVault bool
	confirmStep   bool // true once the first new-vault password has been entered

	passwordInput textinput.Model
	confirmInput  textinput.Model
	searchInput   textinput.Model
	firefoxInput  textinput.Model

	results       []vault.Entry
	cursor        int
	selected      vault.Entry
	reveal        bool
	confirmDelete bool

	entryTitleInput    textinput.Model
	entryURLInput      textinput.Model
	entryUsernameInput textinput.Model
	entryPasswordInput textinput.Model
	entryTOTPInput     textinput.Model
	entryFocus         int
	entryEditing       bool
	entryOriginal      vault.Entry

	err    error
	status string
}

func newManagerState() managerState {
	pw := textinput.New()
	pw.Placeholder = "Vault password"
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'

	confirm := textinput.New()
	confirm.Placeholder = "Confirm vault password"
	confirm.EchoMode = textinput.EchoPassword
	confirm.EchoCharacter = '•'

	search := textinput.New()
	search.Placeholder = "Search by name or URL..."
	search.Focus()

	ff := textinput.New()
	ff.Placeholder = "Firefox master password (leave empty if none)"
	ff.EchoMode = textinput.EchoPassword
	ff.EchoCharacter = '•'

	title := textinput.New()
	title.Placeholder = "Title (e.g. GitHub)"

	url := textinput.New()
	url.Placeholder = "URL"

	username := textinput.New()
	username.Placeholder = "Username"

	password := textinput.New()
	password.Placeholder = "Password"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'

	totpInput := textinput.New()
	totpInput.Placeholder = "TOTP Base32 secret or otpauth:// URI (optional)"
	totpInput.EchoMode = textinput.EchoPassword
	totpInput.EchoCharacter = '•'

	return managerState{
		passwordInput:      pw,
		confirmInput:       confirm,
		searchInput:        search,
		firefoxInput:       ff,
		entryTitleInput:    title,
		entryURLInput:      url,
		entryUsernameInput: username,
		entryPasswordInput: password,
		entryTOTPInput:     totpInput,
	}
}

// enterManager decides whether to jump straight to the list (already
// unlocked this session) or show the unlock/create-vault prompt.
func (m Model) enterManager() (tea.Model, tea.Cmd) {
	if m.manager.vault != nil {
		m.screen = ScreenManagerList
		m.manager.refreshResults()
		return m, nil
	}

	exists, err := vault.Exists()
	if err != nil {
		m.manager.err = err
	}
	m.manager.creatingVault = !exists
	m.manager.confirmStep = false
	m.manager.err = nil
	m.manager.passwordInput.SetValue("")
	m.manager.confirmInput.SetValue("")
	m.manager.passwordInput.Focus()
	m.screen = ScreenManagerUnlock
	return m, nil
}

func (m Model) handleManagerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case ScreenManagerUnlock:
		return m.handleUnlockKey(msg)
	case ScreenManagerFirefoxPassword:
		return m.handleFirefoxPasswordKey(msg)
	case ScreenManagerList:
		return m.handleListKey(msg)
	case ScreenManagerDetail:
		return m.handleDetailKey(msg)
	case ScreenManagerImporting:
		return m, nil // busy; ignore input until the import completes
	case ScreenManagerAddEntry:
		return m.handleAddEntryKey(msg)
	}
	return m, nil
}

func (m Model) updateManager(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case vaultUnlockedMsg:
		m.manager.vault = msg.v
		m.manager.vaultPassword = msg.password
		m.manager.err = nil
		m.screen = ScreenManagerList
		m.manager.refreshResults()
		return m, nil

	case vaultErrorMsg:
		m.manager.err = msg.err
		return m, nil

	case importDoneMsg:
		if msg.err == firefox.ErrWrongMasterPassword {
			m.manager.firefoxInput.SetValue("")
			m.manager.firefoxInput.Focus()
			m.screen = ScreenManagerFirefoxPassword
			return m, nil
		}
		if msg.err != nil {
			m.manager.err = msg.err
			m.screen = ScreenManagerList
			return m, nil
		}
		added, updated := m.manager.vault.MergeImported(msg.entries)
		if err := m.manager.vault.Save(m.manager.vaultPassword); err != nil {
			m.manager.err = fmt.Errorf("saving vault: %w", err)
		} else {
			m.manager.status = fmt.Sprintf("Imported from Firefox: %d new, %d updated (%d skipped)", added, updated, msg.skipped)
			m.manager.err = nil
		}
		m.screen = ScreenManagerList
		m.manager.refreshResults()
		return m, nil
	}
	return m, nil
}

func (m Model) handleUnlockKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = ScreenMenu
		return m, nil

	case "enter":
		if m.manager.creatingVault {
			if !m.manager.confirmStep {
				if m.manager.passwordInput.Value() == "" {
					m.manager.err = fmt.Errorf("vault password cannot be empty")
					return m, nil
				}
				m.manager.confirmStep = true
				m.manager.err = nil
				m.manager.confirmInput.Focus()
				return m, nil
			}
			if m.manager.confirmInput.Value() != m.manager.passwordInput.Value() {
				m.manager.err = fmt.Errorf("passwords do not match")
				m.manager.confirmInput.SetValue("")
				return m, nil
			}
			return m, createVaultCmd(m.manager.passwordInput.Value())
		}
		return m, unlockVaultCmd(m.manager.passwordInput.Value())
	}

	var cmd tea.Cmd
	if m.manager.creatingVault && m.manager.confirmStep {
		m.manager.confirmInput, cmd = m.manager.confirmInput.Update(msg)
	} else {
		m.manager.passwordInput, cmd = m.manager.passwordInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleFirefoxPasswordKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = ScreenManagerList
		return m, nil
	case "enter":
		m.screen = ScreenManagerImporting
		return m, importFirefoxCmd(m.manager.firefoxInput.Value())
	}
	var cmd tea.Cmd
	m.manager.firefoxInput, cmd = m.manager.firefoxInput.Update(msg)
	return m, cmd
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = ScreenMenu
		return m, nil
	case "ctrl+r":
		m.manager.status = ""
		m.manager.err = nil
		m.screen = ScreenManagerImporting
		return m, importFirefoxCmd("")
	case "ctrl+n":
		m.manager.status = ""
		m.manager.err = nil
		m.manager.entryTitleInput.SetValue("")
		m.manager.entryURLInput.SetValue("")
		m.manager.entryUsernameInput.SetValue("")
		m.manager.entryPasswordInput.SetValue("")
		m.manager.entryTOTPInput.SetValue("")
		m.manager.entryEditing = false
		m.manager.entryOriginal = vault.Entry{}
		m.manager.entryFocus = entryFocusTitle
		m.manager.refocusEntryForm()
		m.screen = ScreenManagerAddEntry
		return m, nil
	case "up":
		if m.manager.cursor > 0 {
			m.manager.cursor--
		}
		return m, nil
	case "down":
		if m.manager.cursor < len(m.manager.results)-1 {
			m.manager.cursor++
		}
		return m, nil
	case "enter":
		if len(m.manager.results) > 0 {
			m.manager.selected = m.manager.results[m.manager.cursor]
			m.manager.reveal = false
			m.manager.confirmDelete = false
			m.screen = ScreenManagerDetail
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.manager.searchInput, cmd = m.manager.searchInput.Update(msg)
	m.manager.refreshResults()
	return m, cmd
}

func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "d" {
		if !m.manager.confirmDelete {
			m.manager.confirmDelete = true
			return m, nil
		}
		m.manager.vault.Delete(m.manager.selected.ID)
		if err := m.manager.vault.Save(m.manager.vaultPassword); err != nil {
			m.manager.err = fmt.Errorf("saving vault: %w", err)
		} else {
			m.manager.status = fmt.Sprintf("Deleted %q", m.manager.selected.Title)
			m.manager.err = nil
		}
		m.manager.confirmDelete = false
		m.screen = ScreenManagerList
		m.manager.refreshResults()
		return m, nil
	}
	m.manager.confirmDelete = false

	switch msg.String() {
	case "esc":
		m.screen = ScreenManagerList
		return m, nil
	case "r":
		m.manager.reveal = !m.manager.reveal
		return m, nil
	case "c":
		return m.copySecret(m.manager.selected.Password, "Password")
	case "u":
		return m.copySecret(m.manager.selected.Username, "Username")
	case "t":
		code, err := totp.Code(m.manager.selected.TOTPSecret, time.Now())
		if err != nil {
			m.manager.err = err
			return m, nil
		}
		return m.copySecret(code, "TOTP code")
	case "e":
		m.manager.err = nil
		m.manager.status = ""
		m.manager.entryOriginal = m.manager.selected
		m.manager.entryEditing = true
		m.manager.entryTitleInput.SetValue(m.manager.selected.Title)
		m.manager.entryURLInput.SetValue(m.manager.selected.URL)
		m.manager.entryUsernameInput.SetValue(m.manager.selected.Username)
		m.manager.entryPasswordInput.SetValue(m.manager.selected.Password)
		m.manager.entryTOTPInput.SetValue(m.manager.selected.TOTPSecret)
		m.manager.entryFocus = entryFocusTitle
		m.manager.refocusEntryForm()
		m.screen = ScreenManagerAddEntry
		return m, nil
	}
	return m, nil
}

func (m *managerState) refreshResults() {
	if m.vault == nil {
		m.results = nil
		return
	}
	m.results = m.vault.Search(m.searchInput.Value())
	if m.cursor >= len(m.results) {
		m.cursor = max(0, len(m.results)-1)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// visibleWindow returns the [start, end) slice bounds of the results window
// to render so that cursor stays visible, for a list of the given total
// length capped at maxRows visible rows at a time.
func visibleWindow(cursor, total, maxRows int) (start, end int) {
	if total <= maxRows {
		return 0, total
	}
	start = cursor - maxRows/2
	if start < 0 {
		start = 0
	}
	end = start + maxRows
	if end > total {
		end = total
		start = end - maxRows
	}
	return start, end
}

func (m Model) handleAddEntryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = ScreenManagerList
		return m, nil
	case "up":
		m.manager.entryFocus = (m.manager.entryFocus - 1 + entryFocusCount) % entryFocusCount
		m.manager.refocusEntryForm()
		return m, nil
	case "down":
		m.manager.entryFocus = (m.manager.entryFocus + 1) % entryFocusCount
		m.manager.refocusEntryForm()
		return m, nil
	case "enter":
		if m.manager.entryFocus == entryFocusSave {
			return m.saveNewEntry()
		}
		m.manager.entryFocus = (m.manager.entryFocus + 1) % entryFocusCount
		m.manager.refocusEntryForm()
		return m, nil
	}

	var cmd tea.Cmd
	switch m.manager.entryFocus {
	case entryFocusTitle:
		m.manager.entryTitleInput, cmd = m.manager.entryTitleInput.Update(msg)
	case entryFocusURL:
		m.manager.entryURLInput, cmd = m.manager.entryURLInput.Update(msg)
	case entryFocusUsername:
		m.manager.entryUsernameInput, cmd = m.manager.entryUsernameInput.Update(msg)
	case entryFocusPassword:
		m.manager.entryPasswordInput, cmd = m.manager.entryPasswordInput.Update(msg)
	case entryFocusTOTP:
		m.manager.entryTOTPInput, cmd = m.manager.entryTOTPInput.Update(msg)
	}
	return m, cmd
}

func (m Model) saveNewEntry() (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(m.manager.entryTitleInput.Value())
	url := strings.TrimSpace(m.manager.entryURLInput.Value())
	if title == "" {
		title = url
	}
	if title == "" {
		m.manager.err = fmt.Errorf("title or URL is required")
		return m, nil
	}

	entry := vault.Entry{
		ID:         fmt.Sprintf("manual|%d", time.Now().UnixNano()),
		Title:      title,
		URL:        url,
		Username:   m.manager.entryUsernameInput.Value(),
		Password:   m.manager.entryPasswordInput.Value(),
		TOTPSecret: strings.TrimSpace(m.manager.entryTOTPInput.Value()),
		Source:     "manual",
		UpdatedAt:  time.Now(),
	}
	if m.manager.entryEditing {
		entry.ID = m.manager.entryOriginal.ID
		entry.Source = m.manager.entryOriginal.Source
		entry.Notes = m.manager.entryOriginal.Notes
	}
	if entry.TOTPSecret != "" {
		if _, err := totp.Code(entry.TOTPSecret, time.Now()); err != nil {
			m.manager.err = fmt.Errorf("TOTP: %w", err)
			return m, nil
		}
	}
	m.manager.vault.Upsert(entry)
	if err := m.manager.vault.Save(m.manager.vaultPassword); err != nil {
		m.manager.err = fmt.Errorf("saving vault: %w", err)
		return m, nil
	}

	verb := "Added"
	if m.manager.entryEditing {
		verb = "Updated"
	}
	m.manager.status = fmt.Sprintf("%s %q", verb, entry.Title)
	m.manager.entryEditing = false
	m.manager.entryOriginal = vault.Entry{}
	m.manager.err = nil
	m.screen = ScreenManagerList
	m.manager.refreshResults()
	return m, nil
}

func (m *managerState) refocusEntryForm() {
	m.entryTitleInput.Blur()
	m.entryURLInput.Blur()
	m.entryUsernameInput.Blur()
	m.entryPasswordInput.Blur()
	m.entryTOTPInput.Blur()
	switch m.entryFocus {
	case entryFocusTitle:
		m.entryTitleInput.Focus()
	case entryFocusURL:
		m.entryURLInput.Focus()
	case entryFocusUsername:
		m.entryUsernameInput.Focus()
	case entryFocusPassword:
		m.entryPasswordInput.Focus()
	case entryFocusTOTP:
		m.entryTOTPInput.Focus()
	}
}

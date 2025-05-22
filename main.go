package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
    includeUpper    bool
    includeLower    bool
    includeSymbols  bool
    includeNumbers  bool

    lengthInput     textinput.Model

    password        string

    focused         int
    err             error
}

func initialModel() model {
    ti := textinput.New()
    ti.Placeholder = "Password Length (1-50)"
    ti.Focus()

    return model{
        lengthInput:    ti,
        includeUpper:   false,
        includeLower:   false,
        includeSymbols: false,
        includeNumbers: false,
        focused:        1,
    }
}

func (m model) Init() tea.Cmd {
    return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd

    switch msg := msg.(type) {
        case tea.KeyMsg:
            switch msg.String() {
                case "ctrl+c", "q":
                    return m, tea.Quit

                case "up", "down":
                    if msg.String() == "up" {
                        m.focused = (m.focused - 1 + 6) % 6
                    } else {
                        m.focused = (m.focused + 1) % 6
                    }

                    if m.focused == 1 {
                        m.lengthInput.Focus()
                    } else {
                        m.lengthInput.Blur()
                    }

                case " ":
                    switch m.focused {
                        case 2:
                            m.includeUpper = !m.includeUpper
                        case 3:
                            m.includeLower = !m.includeLower
                        case 4:
                            m.includeSymbols = !m.includeSymbols
                        case 5:
                            m.includeNumbers = !m.includeNumbers
                    }

                case "enter":
                    if m.focused == 0 {
                        m.generatePassword()
                    }
            }

            if m.focused == 1 {
                m.lengthInput, cmd = m.lengthInput.Update(msg)
            }
    }

    return m, cmd
}

func (m *model) generatePassword() {
    m.err = nil
    m.password = ""

    length, err := strconv.Atoi(m.lengthInput.Value())
    if err != nil || length <= 0 || length > 50 {
        m.err = fmt.Errorf("length must be a number between 1 and 50")
        return
    }

    var charset string
    if m.includeUpper {
        charset += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    }
    if m.includeLower {
        charset += "abcdefghijklmnopqrstuvwxyz"
    }
    if m.includeSymbols {
        charset += "!@#$%^&*()-_=+[]{}|;:,.<>/?"
    }
    if m.includeNumbers {
        charset += "0123456789"
    }

    if charset == "" {
        m.err = fmt.Errorf("select at least one character set")
        return
    }

    password, err := generatePassword(length, charset)
    if err != nil {
        m.err = err
        return
    }

    m.password = password
}

func (m model) View() string {
    titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7")).Bold(true)
    focusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f5e0dc"))
    normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
    checkboxStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7f849c"))
    selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))
    errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))

    renderCheckbox := func(label string, checked bool, focused bool) string {
        checkbox := "[ ]"
        if checked {
            checkbox = "[x]"
        }

        if focused {
            return focusedStyle.Render(fmt.Sprintf("%s %s", checkbox, label))
        }

        if checked {
            return selectedStyle.Render(fmt.Sprintf("%s %s", checkbox, label))
        }

        return checkboxStyle.Render(fmt.Sprintf("%s %s", checkbox, label))
    }

    view := titleStyle.Render("Password Generator TUI") + "\n\n"

    lengthLabel := "Password Length: "
    if m.focused == 1 {
        lengthLabel = focusedStyle.Render("> " + lengthLabel)
    } else {
        lengthLabel = normalStyle.Render("  " + lengthLabel)
    }
    view += lengthLabel + m.lengthInput.View() + "\n\n"

    view += renderCheckbox("Uppercase", m.includeUpper, m.focused == 2) + "\n"
    view += renderCheckbox("Lowercase", m.includeLower, m.focused == 3) + "\n"
    view += renderCheckbox("Symbols", m.includeSymbols, m.focused == 4) + "\n"
    view += renderCheckbox("Numbers", m.includeNumbers, m.focused == 5) + "\n\n"

    generateLabel := "Generate Password"
    if m.focused == 0 {
        generateLabel = focusedStyle.Render("> " + generateLabel)
    } else {
        generateLabel = normalStyle.Render("  " + generateLabel)
    }
    view += generateLabel + "\n\n"

    if m.err != nil {
        view += errorStyle.Render("Error: " + m.err.Error()) + "\n\n"
    }

    if m.password != "" {
        view += "Generated Password: " + selectedStyle.Render(m.password) + "\n"
    }

    view += "\n\nUse arrow keys to navigate, space to toggle, enter to generate, 'q' to quit"

    return view
}

func generatePassword(length int, charset string) (string, error) {
    password := make([]byte, length)
    for i := range password {
        index, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
        if err != nil {
            return "", err
        }
        password[i] = charset[index.Int64()]
    }
    return string(password), nil
}

func main() {
    p := tea.NewProgram(initialModel())
    if _, err := p.Run(); err != nil {
        fmt.Println("error running program:", err)
        os.Exit(1)
    }
}
package ui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hussaratkuro/gopass/internal/generator"
)

// generatorState holds the state for the password generator screen.
type generatorState struct {
	opts generator.Options

	lengthInput textinput.Model
	password    string
	focused     int // 0=generate button, 1=length input, 2..5=checkboxes
	err         error
}

const (
	genFocusGenerate = 0
	genFocusLength   = 1
	genFocusUpper    = 2
	genFocusLower    = 3
	genFocusSymbols  = 4
	genFocusNumbers  = 5
	genFocusCount    = 6
)

func newGeneratorState() generatorState {
	ti := textinput.New()
	ti.Placeholder = "Password Length (1-50)"
	ti.Focus()
	return generatorState{
		lengthInput: ti,
		focused:     genFocusLength,
	}
}

func (m Model) updateGeneratorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	g := &m.generator
	var cmd tea.Cmd

	switch msg.String() {
	case "up":
		g.focused = (g.focused - 1 + genFocusCount) % genFocusCount
		g.refocus()
	case "down":
		g.focused = (g.focused + 1) % genFocusCount
		g.refocus()
	case " ":
		switch g.focused {
		case genFocusUpper:
			g.opts.Upper = !g.opts.Upper
		case genFocusLower:
			g.opts.Lower = !g.opts.Lower
		case genFocusSymbols:
			g.opts.Symbols = !g.opts.Symbols
		case genFocusNumbers:
			g.opts.Numbers = !g.opts.Numbers
		}
	case "enter":
		if g.focused == genFocusGenerate {
			g.generate()
		}
	}

	if g.focused == genFocusLength {
		g.lengthInput, cmd = g.lengthInput.Update(msg)
	}
	return m, cmd
}

func (g *generatorState) refocus() {
	if g.focused == genFocusLength {
		g.lengthInput.Focus()
	} else {
		g.lengthInput.Blur()
	}
}

func (g *generatorState) generate() {
	g.err = nil
	g.password = ""

	length, err := strconv.Atoi(g.lengthInput.Value())
	if err != nil {
		g.err = fmt.Errorf("length must be a number between %d and %d", generator.MinLength, generator.MaxLength)
		return
	}

	password, err := generator.Generate(length, g.opts)
	if err != nil {
		g.err = err
		return
	}
	g.password = password
}

func viewGenerator(g generatorState) string {
	renderCheckbox := func(label string, checked bool, focused bool) string {
		checkbox := "[ ]"
		if checked {
			checkbox = "[x]"
		}
		switch {
		case focused:
			return FocusedStyle.Render(fmt.Sprintf("%s %s", checkbox, label))
		case checked:
			return SelectedStyle.Render(fmt.Sprintf("%s %s", checkbox, label))
		default:
			return MutedStyle.Render(fmt.Sprintf("%s %s", checkbox, label))
		}
	}

	view := TitleStyle.Render("Password Generator") + "\n\n"

	lengthLabel := "Password Length: "
	if g.focused == genFocusLength {
		lengthLabel = FocusedStyle.Render("> " + lengthLabel)
	} else {
		lengthLabel = NormalStyle.Render("  " + lengthLabel)
	}
	view += lengthLabel + g.lengthInput.View() + "\n\n"

	view += renderCheckbox("Uppercase", g.opts.Upper, g.focused == genFocusUpper) + "\n"
	view += renderCheckbox("Lowercase", g.opts.Lower, g.focused == genFocusLower) + "\n"
	view += renderCheckbox("Symbols", g.opts.Symbols, g.focused == genFocusSymbols) + "\n"
	view += renderCheckbox("Numbers", g.opts.Numbers, g.focused == genFocusNumbers) + "\n\n"

	generateLabel := "Generate Password"
	if g.focused == genFocusGenerate {
		generateLabel = FocusedStyle.Render("> " + generateLabel)
	} else {
		generateLabel = NormalStyle.Render("  " + generateLabel)
	}
	view += generateLabel + "\n\n"

	if g.err != nil {
		view += ErrorStyle.Render("Error: "+g.err.Error()) + "\n\n"
	}

	if g.password != "" {
		view += "Generated Password: " + SelectedStyle.Render(g.password) + "\n"
	}

	view += "\n" + HelpStyle.Render("↑/↓ navigate · space toggle · enter generate · esc back") + "\n"
	return lipgloss.NewStyle().Render(view)
}

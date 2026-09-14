package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/hussaratkuro/gopass/internal/credentialcli"
	"github.com/hussaratkuro/gopass/internal/ui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "credential" {
		if err := credentialcli.Run(os.Args[2:], os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "gopass credential:", err)
			os.Exit(1)
		}
		return
	}
	p := tea.NewProgram(ui.New())
	if _, err := p.Run(); err != nil {
		fmt.Println("error running program:", err)
		os.Exit(1)
	}
}

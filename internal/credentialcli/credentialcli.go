package credentialcli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hussaratkuro/gopass/internal/totp"
	"github.com/hussaratkuro/gopass/internal/vault"
)

const Usage = `gopass credential - encrypted credential provider

Usage:
  gopass credential list --password-stdin
  gopass credential get --ref ID_OR_TITLE --password-stdin
  gopass credential get --ref ID_OR_TITLE --field password --password-stdin
  gopass credential get --ref ID_OR_TITLE --field totp --password-stdin

The vault password is read as one line from stdin. It is never accepted as a
command-line argument or environment variable.
`

type Summary struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url,omitempty"`
	Username string `json:"username,omitempty"`
}

type Credential struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password"`
	TOTP     string `json:"totp,omitempty"`
}

func Run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		_, err := io.WriteString(stdout, Usage)
		return err
	}
	action := args[0]
	ref, field, passwordStdin, err := parseOptions(args[1:])
	if err != nil {
		return err
	}
	if !passwordStdin {
		return errors.New("--password-stdin is required")
	}
	password, err := readPassword(stdin)
	if err != nil {
		return err
	}
	store, err := vault.Load(password)
	if err != nil {
		return fmt.Errorf("unlock vault: %w", err)
	}

	switch action {
	case "list":
		if ref != "" || field != "" {
			return errors.New("list does not accept --ref or --field")
		}
		entries := store.Search("")
		summaries := make([]Summary, 0, len(entries))
		for _, entry := range entries {
			summaries = append(summaries, Summary{
				ID: entry.ID, Title: entry.Title, URL: entry.URL, Username: entry.Username,
			})
		}
		return json.NewEncoder(stdout).Encode(summaries)
	case "get":
		if strings.TrimSpace(ref) == "" {
			return errors.New("get requires --ref ID_OR_TITLE")
		}
		entry, err := resolve(store.Entries, ref)
		if err != nil {
			return err
		}
		credential := Credential{
			ID: entry.ID, Title: entry.Title, URL: entry.URL,
			Username: entry.Username, Password: entry.Password,
		}
		if entry.TOTPSecret != "" {
			credential.TOTP, err = totp.Code(entry.TOTPSecret, time.Now())
			if err != nil {
				return fmt.Errorf("generate TOTP: %w", err)
			}
		}
		switch field {
		case "":
			return json.NewEncoder(stdout).Encode(credential)
		case "id":
			_, err = fmt.Fprintln(stdout, credential.ID)
		case "title":
			_, err = fmt.Fprintln(stdout, credential.Title)
		case "url":
			_, err = fmt.Fprintln(stdout, credential.URL)
		case "username":
			_, err = fmt.Fprintln(stdout, credential.Username)
		case "password":
			_, err = fmt.Fprintln(stdout, credential.Password)
		case "totp":
			if credential.TOTP == "" {
				return errors.New("credential has no TOTP secret")
			}
			_, err = fmt.Fprintln(stdout, credential.TOTP)
		default:
			return fmt.Errorf("unknown field %q", field)
		}
		return err
	default:
		return fmt.Errorf("unknown credential action %q", action)
	}
}

func parseOptions(args []string) (ref, field string, passwordStdin bool, err error) {
	for index := 0; index < len(args); index++ {
		switch arg := args[index]; {
		case arg == "--password-stdin":
			passwordStdin = true
		case arg == "--ref" || arg == "--field":
			if index+1 >= len(args) {
				return "", "", false, fmt.Errorf("%s requires a value", arg)
			}
			index++
			if arg == "--ref" {
				ref = args[index]
			} else {
				field = strings.ToLower(args[index])
			}
		case strings.HasPrefix(arg, "--ref="):
			ref = strings.TrimPrefix(arg, "--ref=")
		case strings.HasPrefix(arg, "--field="):
			field = strings.ToLower(strings.TrimPrefix(arg, "--field="))
		default:
			return "", "", false, fmt.Errorf("unknown option %q", arg)
		}
	}
	return ref, field, passwordStdin, nil
}

func readPassword(input io.Reader) (string, error) {
	password, err := bufio.NewReader(io.LimitReader(input, 16*1024)).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read vault password: %w", err)
	}
	password = strings.TrimSuffix(password, "\n")
	password = strings.TrimSuffix(password, "\r")
	if password == "" {
		return "", errors.New("vault password cannot be empty")
	}
	return password, nil
}

func resolve(entries []vault.Entry, ref string) (vault.Entry, error) {
	ref = strings.TrimSpace(ref)
	for _, entry := range entries {
		if entry.ID == ref {
			return entry, nil
		}
	}
	var match *vault.Entry
	for index := range entries {
		if !strings.EqualFold(strings.TrimSpace(entries[index].Title), ref) {
			continue
		}
		if match != nil {
			return vault.Entry{}, fmt.Errorf("credential title %q is ambiguous; use its ID", ref)
		}
		entry := entries[index]
		match = &entry
	}
	if match == nil {
		return vault.Entry{}, fmt.Errorf("credential %q not found", ref)
	}
	return *match, nil
}

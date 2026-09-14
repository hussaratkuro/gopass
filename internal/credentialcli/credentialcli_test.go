package credentialcli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hussaratkuro/gopass/internal/vault"
)

func TestListAndGetCredential(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store := vault.New()
	store.Upsert(vault.Entry{ID: "nas-id", Title: "Office NAS", URL: "ftpes://nas.example", Username: "alice", Password: "secret"})
	if err := store.Save("master password"); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := Run([]string{"list", "--password-stdin"}, strings.NewReader("master password\n"), &output); err != nil {
		t.Fatal(err)
	}
	var summaries []Summary
	if err := json.Unmarshal(output.Bytes(), &summaries); err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].ID != "nas-id" || strings.Contains(output.String(), "secret") {
		t.Fatalf("credential summaries = %#v, raw %q", summaries, output.String())
	}

	output.Reset()
	if err := Run([]string{"get", "--ref", "office nas", "--password-stdin"}, strings.NewReader("master password\n"), &output); err != nil {
		t.Fatal(err)
	}
	var credential Credential
	if err := json.Unmarshal(output.Bytes(), &credential); err != nil {
		t.Fatal(err)
	}
	if credential.ID != "nas-id" || credential.Username != "alice" || credential.Password != "secret" {
		t.Fatalf("credential = %#v", credential)
	}
}

func TestCredentialCLIRejectsUnsafeOrAmbiguousRequests(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store := vault.New()
	store.Upsert(vault.Entry{ID: "one", Title: "Duplicate", Password: "first"})
	store.Upsert(vault.Entry{ID: "two", Title: "duplicate", Password: "second"})
	if err := store.Save("master"); err != nil {
		t.Fatal(err)
	}

	if err := Run([]string{"get", "--ref", "one"}, strings.NewReader("master\n"), &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "--password-stdin") {
		t.Fatalf("missing stdin protection error = %v", err)
	}
	if err := Run([]string{"get", "--ref", "duplicate", "--password-stdin"}, strings.NewReader("master\n"), &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous title error = %v", err)
	}
	if err := Run([]string{"get", "--ref", "one", "--password-stdin"}, strings.NewReader("wrong\n"), &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "unlock vault") {
		t.Fatalf("wrong password error = %v", err)
	}
}

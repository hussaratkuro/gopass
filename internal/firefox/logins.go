package firefox

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// Credential is one decrypted Firefox saved login.
type Credential struct {
	Hostname string // e.g. https://example.com
	Username string
	Password string
}

// DisplayName returns a short, human-friendly label for the credential,
// derived from its hostname (e.g. "example.com").
func (c Credential) DisplayName() string {
	if u, err := url.Parse(c.Hostname); err == nil && u.Host != "" {
		return u.Host
	}
	return c.Hostname
}

type rawLogin struct {
	Hostname          string `json:"hostname"`
	EncryptedUsername string `json:"encryptedUsername"`
	EncryptedPassword string `json:"encryptedPassword"`
}

type loginsFile struct {
	Logins []rawLogin `json:"logins"`
}

// ImportResult is the outcome of decrypting a profile's saved logins.
type ImportResult struct {
	Credentials []Credential
	Skipped     int // entries that failed to decrypt (reported, not fatal)
}

// Import decrypts and returns every saved login in the given Firefox
// profile directory. masterPassword may be empty if the profile has no
// master password set.
func Import(profileDir, masterPassword string) (ImportResult, error) {
	keys, err := OpenProfileKeys(profileDir, masterPassword)
	if err != nil {
		return ImportResult{}, err
	}

	raw, err := os.ReadFile(filepath.Join(profileDir, "logins.json"))
	if err != nil {
		return ImportResult{}, fmt.Errorf("reading logins.json: %w", err)
	}
	var lf loginsFile
	if err := json.Unmarshal(raw, &lf); err != nil {
		return ImportResult{}, fmt.Errorf("parsing logins.json: %w", err)
	}

	// All entries in a profile are encrypted with the same active key, so
	// once one candidate proves correct it's tried first for every
	// subsequent field; a full scan remains as a fallback.
	active := 0

	decrypt := func(b64 string) (string, error) {
		if s, err := DecryptField(keys[active], b64); err == nil {
			return s, nil
		}
		for i, k := range keys {
			if i == active {
				continue
			}
			if s, err := DecryptField(k, b64); err == nil {
				active = i
				return s, nil
			}
		}
		return "", fmt.Errorf("no candidate key decrypted this field")
	}

	var result ImportResult
	var firstErr error
	for _, l := range lf.Logins {
		if l.EncryptedUsername == "" || l.EncryptedPassword == "" {
			continue
		}
		user, err := decrypt(l.EncryptedUsername)
		if err != nil {
			result.Skipped++
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", l.Hostname, err)
			}
			continue
		}
		pass, err := decrypt(l.EncryptedPassword)
		if err != nil {
			result.Skipped++
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", l.Hostname, err)
			}
			continue
		}
		result.Credentials = append(result.Credentials, Credential{
			Hostname: l.Hostname,
			Username: user,
			Password: pass,
		})
	}

	if len(result.Credentials) == 0 && firstErr != nil {
		return ImportResult{}, fmt.Errorf("failed to decrypt any login; first error: %w", firstErr)
	}
	return result, nil
}

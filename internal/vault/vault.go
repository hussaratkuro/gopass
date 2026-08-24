// Package vault implements a local, password-encrypted store of credentials.
//
// The vault file is never written in plaintext: its contents are serialized
// to JSON, then sealed with AES-256-GCM using a key derived from the vault
// password via Argon2id. The vault password is independent from any Firefox
// master password used during import.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// ErrWrongPassword is returned by Load when the vault cannot be decrypted
// with the supplied password.
var ErrWrongPassword = errors.New("wrong vault password")

const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // KiB
	argonThreads = 4
	keyLen       = 32
	saltLen      = 16
)

// Entry is a single stored credential.
type Entry struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Notes     string    `json:"notes"`
	Source    string    `json:"source"` // "firefox" or "manual"
	UpdatedAt time.Time `json:"updated_at"`
}

// Vault is an in-memory, decrypted collection of entries.
type Vault struct {
	Entries []Entry `json:"entries"`
}

type envelope struct {
	Version int    `json:"version"`
	Salt    string `json:"salt"`
	Nonce   string `json:"nonce"`
	Data    string `json:"data"`
}

// Path returns the on-disk location of the vault file, creating its parent
// directory if necessary.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "gopass")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "vault.json"), nil
}

// Exists reports whether a vault file has already been created.
func Exists() (bool, error) {
	path, err := Path()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, keyLen)
}

// New creates an empty, unsaved vault.
func New() *Vault {
	return &Vault{}
}

// Load reads and decrypts the vault file using password.
func Load(password string) (*Vault, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("corrupt vault file: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(env.Salt)
	if err != nil {
		return nil, fmt.Errorf("corrupt vault file: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil {
		return nil, fmt.Errorf("corrupt vault file: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(env.Data)
	if err != nil {
		return nil, fmt.Errorf("corrupt vault file: %w", err)
	}

	key := deriveKey(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, ErrWrongPassword
	}

	var v Vault
	if err := json.Unmarshal(plain, &v); err != nil {
		return nil, fmt.Errorf("corrupt vault contents: %w", err)
	}
	return &v, nil
}

// Save encrypts and writes the vault to disk under password, replacing any
// existing vault file. A fresh random salt and nonce are used every time.
func (v *Vault) Save(password string) error {
	path, err := Path()
	if err != nil {
		return err
	}

	plain, err := json.Marshal(v)
	if err != nil {
		return err
	}

	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	sealed := gcm.Seal(nil, nonce, plain, nil)

	env := envelope{
		Version: 1,
		Salt:    base64.StdEncoding.EncodeToString(salt),
		Nonce:   base64.StdEncoding.EncodeToString(nonce),
		Data:    base64.StdEncoding.EncodeToString(sealed),
	}
	out, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
}

// Upsert inserts e, or replaces an existing entry with the same ID.
func (v *Vault) Upsert(e Entry) {
	for i := range v.Entries {
		if v.Entries[i].ID == e.ID {
			v.Entries[i] = e
			return
		}
	}
	v.Entries = append(v.Entries, e)
}

// Delete removes the entry with the given ID, reporting whether one was found.
func (v *Vault) Delete(id string) bool {
	for i := range v.Entries {
		if v.Entries[i].ID == id {
			v.Entries = append(v.Entries[:i], v.Entries[i+1:]...)
			return true
		}
	}
	return false
}

// MergeImported adds entries from an import pass, keyed by Source+URL+Username
// so re-running an import updates existing entries instead of duplicating them.
func (v *Vault) MergeImported(entries []Entry) (added, updated int) {
	index := make(map[string]int, len(v.Entries))
	for i, e := range v.Entries {
		index[importKey(e)] = i
	}
	for _, e := range entries {
		if i, ok := index[importKey(e)]; ok {
			v.Entries[i] = e
			updated++
		} else {
			v.Entries = append(v.Entries, e)
			index[importKey(e)] = len(v.Entries) - 1
			added++
		}
	}
	return added, updated
}

func importKey(e Entry) string {
	return e.Source + "|" + e.URL + "|" + e.Username
}

// Search returns entries whose title, URL, or username contain query
// (case-insensitive), sorted by title.
func (v *Vault) Search(query string) []Entry {
	query = strings.ToLower(strings.TrimSpace(query))
	var out []Entry
	for _, e := range v.Entries {
		if query == "" ||
			strings.Contains(strings.ToLower(e.Title), query) ||
			strings.Contains(strings.ToLower(e.URL), query) ||
			strings.Contains(strings.ToLower(e.Username), query) {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title)
	})
	return out
}

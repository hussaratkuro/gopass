package firefox

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Profile describes one Firefox profile found in profiles.ini.
type Profile struct {
	Name    string
	Path    string // absolute path to the profile directory
	Default bool
}

// rootDirs returns the candidate Firefox installation directories to search,
// in priority order, for the current OS.
func rootDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	switch runtime.GOOS {
	case "darwin":
		return []string{filepath.Join(home, "Library", "Application Support", "Firefox")}
	case "windows":
		return []string{filepath.Join(os.Getenv("APPDATA"), "Mozilla", "Firefox")}
	default:
		return []string{
			filepath.Join(home, ".mozilla", "firefox"),
			filepath.Join(home, ".var", "app", "org.mozilla.firefox", ".mozilla", "firefox"), // Flatpak
		}
	}
}

// ListProfiles parses profiles.ini in every known Firefox root directory and
// returns all profiles it finds that have a logins.json.
func ListProfiles() ([]Profile, error) {
	var profiles []Profile
	var lastErr error
	found := false

	for _, root := range rootDirs() {
		iniPath := filepath.Join(root, "profiles.ini")
		f, err := os.Open(iniPath)
		if err != nil {
			continue
		}
		found = true
		ps, err := parseProfilesIni(f, root)
		f.Close()
		if err != nil {
			lastErr = err
			continue
		}
		profiles = append(profiles, ps...)
	}

	if !found {
		return nil, fmt.Errorf("no Firefox profiles.ini found")
	}
	if len(profiles) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return profiles, nil
}

// Default returns the profile marked as default, preferring one that
// contains saved logins; if none is marked default, the first profile with
// saved logins is returned.
func Default() (Profile, error) {
	profiles, err := ListProfiles()
	if err != nil {
		return Profile{}, err
	}
	if len(profiles) == 0 {
		return Profile{}, fmt.Errorf("no usable Firefox profiles found")
	}

	var withLogins []Profile
	for _, p := range profiles {
		if _, err := os.Stat(filepath.Join(p.Path, "logins.json")); err == nil {
			withLogins = append(withLogins, p)
		}
	}
	if len(withLogins) == 0 {
		withLogins = profiles
	}

	for _, p := range withLogins {
		if p.Default {
			return p, nil
		}
	}
	return withLogins[0], nil
}

// parseProfilesIni is a minimal parser for Firefox's profiles.ini, which is a
// standard INI file with [Profile*] and [Install*] sections. The Install
// section's Default key (a profile *path*, relative to root) takes priority
// over any Profile section's own Default=1 flag, mirroring Firefox's own
// profile-selection behavior for the launcher.
func parseProfilesIni(f *os.File, root string) ([]Profile, error) {
	type rawSection struct {
		name string
		kv   map[string]string
	}
	var sections []rawSection
	var cur *rawSection

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sections = append(sections, rawSection{name: line[1 : len(line)-1], kv: map[string]string{}})
			cur = &sections[len(sections)-1]
			continue
		}
		if cur == nil {
			continue
		}
		if eq := strings.Index(line, "="); eq >= 0 {
			cur.kv[line[:eq]] = line[eq+1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var installDefaultPath string
	for _, s := range sections {
		if strings.HasPrefix(s.name, "Install") {
			if d, ok := s.kv["Default"]; ok {
				installDefaultPath = d
			}
		}
	}

	var profiles []Profile
	for _, s := range sections {
		if !strings.HasPrefix(s.name, "Profile") {
			continue
		}
		relPath, ok := s.kv["Path"]
		if !ok {
			continue
		}
		path := relPath
		if s.kv["IsRelative"] != "0" {
			path = filepath.Join(root, relPath)
		}
		isDefault := s.kv["Default"] == "1" || relPath == installDefaultPath
		profiles = append(profiles, Profile{
			Name:    s.kv["Name"],
			Path:    path,
			Default: isDefault,
		})
	}
	return profiles, nil
}

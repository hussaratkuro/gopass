# gopass

An encrypted terminal password manager and generator. The manager supports
manual entries, Firefox import, editing, TOTP codes, and fuzzy commands with
`Ctrl+Shift+P` (`Ctrl+P` fallback).

From an entry detail screen:

- `e` edits the entry, including its optional Base32/`otpauth://` TOTP secret;
- `c`, `u`, and `t` copy password, username, or the current TOTP code;
- copied secrets are cleared after 30 seconds only if the clipboard still
  contains that same value.

Other local tools can read a credential without putting the vault password in
arguments or environment variables:

```text
gopass credential list --password-stdin
gopass credential get --ref ID_OR_UNIQUE_TITLE --password-stdin
gopass credential get --ref ID_OR_UNIQUE_TITLE --field password --password-stdin
gopass credential get --ref ID_OR_UNIQUE_TITLE --field totp --password-stdin
```

The vault password is one line on stdin. JSON list output never contains
passwords, and full credential output never exposes the stored TOTP secret.

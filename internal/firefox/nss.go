package firefox

// This file implements just enough of Mozilla's NSS "Secret Decoder Ring"
// scheme to decrypt the credentials Firefox stores in key4.db + logins.json.
// There is no official Go (or even NSS-external) API for this; the approach
// below follows the algorithm documented and maintained by the firepwd
// project (https://github.com/lclevy/firepwd), the closest thing to a
// reference implementation, cross-checked against a real profile.
//
// key4.db (SQLite) holds:
//   - metadata.item1/item2: a global salt and a PBE-wrapped "password-check"
//     value, used only to verify the (possibly empty) master password.
//   - nssPrivate.a11/a102: PBE-wrapped key material. Firefox profiles that
//     have gone through a Sync key rotation can contain *multiple* rows
//     sharing the same key ID (a102); only one of them decrypts to a
//     validly-padded key, so every candidate is tried.
//
// The PBE wrapping is either the legacy pbeWithSha1AndTripleDES-CBC scheme,
// or (modern NSS) PBES2 with PBKDF2/HMAC-SHA256 and AES-256-CBC. Which one
// is used can differ between the key-check value and the actual key, and
// independently, individual logins.json fields carry their own algorithm
// OID (3DES or AES-256) that must be honored per field.

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"fmt"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"
	_ "modernc.org/sqlite"
)

// ErrWrongMasterPassword is returned when the supplied Firefox master
// password (or its absence) fails the profile's password-check value.
var ErrWrongMasterPassword = errors.New("wrong Firefox master password")

const (
	oidPBEWithSHA1AndTripleDESCBC = "1.2.840.113549.1.12.5.1.3"
	oidPBES2                      = "1.2.840.113549.1.5.13"
	oidPBKDF2                     = "1.2.840.113549.1.5.12"
	oidAES256CBC                  = "2.16.840.1.101.3.4.1.42"
	oidDESEDE3CBC                 = "1.2.840.113549.3.7"

	passwordCheckPlaintext = "password-check\x02\x02"
)

// cka_id is the fixed 16-byte key identifier NSS uses for the
// login-encryption key: 0xf8, fourteen zero bytes, then 0x01.
var ckaID = append(append([]byte{0xf8}, make([]byte, 14)...), 0x01)

// Key is the derived symmetric key used to decrypt individual login fields.
type Key struct {
	bytes []byte // unpadded; 24 bytes (3DES) or 32 bytes (AES-256)
}

type algIdentifier struct {
	OID    asn1.ObjectIdentifier
	Params asn1.RawValue `asn1:"optional"`
}

type pbeEnvelope struct {
	Algorithm  asn1.RawValue // re-decoded into algIdentifier once the OID is known
	Ciphertext []byte
}

type tripleDESParams struct {
	EntrySalt  []byte
	Iterations int
}

type pbkdf2Params struct {
	Salt           []byte
	IterationCount int
	KeyLength      int           `asn1:"optional"`
	PRF            asn1.RawValue `asn1:"optional"`
}

type pbes2Params struct {
	KeyDerivationFunc algIdentifier
	EncryptionScheme  algIdentifier
}

// OpenProfileKeys opens the NSS key database in profileDir and derives every
// candidate key usable to decrypt saved logins, trying masterPassword (empty
// for profiles with no master password set).
//
// Profiles that have been through a Firefox Sync key rotation can carry
// several nssPrivate rows tagged with the same key ID, more than one of
// which decrypts to something with self-consistent PKCS#7 padding — but
// only one of them is the key actually used to encrypt the current
// logins.json. Padding validity alone can't tell them apart, so every
// self-consistent candidate is returned and the caller must try each one
// against real field ciphertext (see DecryptField) to find the real key.
func OpenProfileKeys(profileDir, masterPassword string) ([]*Key, error) {
	dsn := fmt.Sprintf("file:%s?mode=ro", filepath.Join(profileDir, "key4.db"))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening key4.db: %w", err)
	}
	defer db.Close()

	var globalSalt, item2 []byte
	row := db.QueryRow(`SELECT item1, item2 FROM metadata WHERE id = 'password'`)
	if err := row.Scan(&globalSalt, &item2); err != nil {
		return nil, fmt.Errorf("reading key4.db metadata: %w", err)
	}

	pw := []byte(masterPassword)
	check, err := decryptPBE(item2, pw, globalSalt)
	if err != nil {
		return nil, fmt.Errorf("decrypting password-check: %w", err)
	}
	if string(check) != passwordCheckPlaintext {
		return nil, ErrWrongMasterPassword
	}

	rows, err := db.Query(`SELECT a11, a102 FROM nssPrivate`)
	if err != nil {
		return nil, fmt.Errorf("reading key4.db nssPrivate: %w", err)
	}
	defer rows.Close()

	var keys []*Key
	for rows.Next() {
		var a11, a102 []byte
		if err := rows.Scan(&a11, &a102); err != nil {
			return nil, err
		}
		if a11 == nil || !bytesEqual(a102, ckaID) {
			continue
		}
		raw, err := decryptPBE(a11, pw, globalSalt)
		if err != nil {
			continue // try the next candidate row
		}
		if key, ok := stripPKCS7(raw, 16); ok && (len(key) == 24 || len(key) == 32) {
			keys = append(keys, &Key{bytes: key})
			continue
		}
		// legacy 3DES wrap pads to an 8-byte boundary instead.
		if key, ok := stripPKCS7(raw, 8); ok && (len(key) == 24 || len(key) == 32) {
			keys = append(keys, &Key{bytes: key})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("could not derive a usable key from key4.db (no valid nssPrivate entry)")
	}
	return keys, nil
}

// decryptPBE decrypts a PBE-wrapped ASN.1 blob (metadata.item2 or
// nssPrivate.a11 shape): SEQUENCE{ SEQUENCE{OID, params}, OCTETSTRING ct }.
func decryptPBE(blob, masterPassword, globalSalt []byte) ([]byte, error) {
	var env pbeEnvelope
	if _, err := asn1.Unmarshal(blob, &env); err != nil {
		return nil, fmt.Errorf("parsing PBE envelope: %w", err)
	}

	var alg algIdentifier
	if _, err := asn1.Unmarshal(env.Algorithm.FullBytes, &alg); err != nil {
		return nil, fmt.Errorf("parsing PBE algorithm: %w", err)
	}

	switch alg.OID.String() {
	case oidPBEWithSHA1AndTripleDESCBC:
		var params tripleDESParams
		if _, err := asn1.Unmarshal(alg.Params.FullBytes, &params); err != nil {
			return nil, fmt.Errorf("parsing 3DES PBE params: %w", err)
		}
		return decryptMoz3DES(globalSalt, masterPassword, params.EntrySalt, env.Ciphertext)

	case oidPBES2:
		var params pbes2Params
		if _, err := asn1.Unmarshal(alg.Params.FullBytes, &params); err != nil {
			return nil, fmt.Errorf("parsing PBES2 params: %w", err)
		}
		if params.KeyDerivationFunc.OID.String() != oidPBKDF2 {
			return nil, fmt.Errorf("unsupported PBES2 KDF %s", params.KeyDerivationFunc.OID)
		}
		if params.EncryptionScheme.OID.String() != oidAES256CBC {
			return nil, fmt.Errorf("unsupported PBES2 cipher %s", params.EncryptionScheme.OID)
		}
		var kdfParams pbkdf2Params
		if _, err := asn1.Unmarshal(params.KeyDerivationFunc.Params.FullBytes, &kdfParams); err != nil {
			return nil, fmt.Errorf("parsing PBKDF2 params: %w", err)
		}
		keyLen := kdfParams.KeyLength
		if keyLen == 0 {
			keyLen = 32
		}

		// NSS derives the PBKDF2 "password" as SHA1(globalSalt || masterPassword).
		mixed := sha1.Sum(append(append([]byte{}, globalSalt...), masterPassword...))
		key := pbkdf2.Key(mixed[:], kdfParams.Salt, kdfParams.IterationCount, keyLen, sha256.New)

		// The encryption scheme's IV is a raw OCTETSTRING *content*; NSS
		// re-wraps it with its own tag+length header before use as the IV.
		ivRaw := params.EncryptionScheme.Params.Bytes
		iv := append([]byte{0x04, byte(len(ivRaw))}, ivRaw...)

		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		if len(env.Ciphertext)%block.BlockSize() != 0 || len(iv) != block.BlockSize() {
			return nil, fmt.Errorf("malformed PBES2 ciphertext/iv")
		}
		out := make([]byte, len(env.Ciphertext))
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, env.Ciphertext)
		return out, nil

	default:
		return nil, fmt.Errorf("unsupported PBE algorithm %s", alg.OID)
	}
}

// decryptMoz3DES implements NSS's legacy (pre-PBES2) key-wrap scheme; see
// http://www.drh-consultancy.demon.co.uk/key3.html.
func decryptMoz3DES(globalSalt, masterPassword, entrySalt, ciphertext []byte) ([]byte, error) {
	hp := sha1.Sum(append(append([]byte{}, globalSalt...), masterPassword...))
	pes := append(append([]byte{}, entrySalt...), make([]byte, max(0, 20-len(entrySalt)))...)
	chp := sha1.Sum(append(hp[:], entrySalt...))

	k1 := hmacSHA1(chp[:], append(append([]byte{}, pes...), entrySalt...))
	tk := hmacSHA1(chp[:], pes)
	k2 := hmacSHA1(chp[:], append(tk, entrySalt...))
	k := append(k1, k2...)

	iv := k[len(k)-8:]
	key := k[:24]

	block, err := des.NewTripleDESCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("malformed 3DES ciphertext")
	}
	out := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, ciphertext)
	return out, nil
}

func hmacSHA1(key, data []byte) []byte {
	h := hmac.New(sha1.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// DecryptField decrypts one base64-encoded logins.json field
// (encryptedUsername/encryptedPassword) using key.
func DecryptField(key *Key, b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}

	var field struct {
		KeyID     []byte
		Algorithm algIdentifier
		Cipher    []byte
	}
	if _, err := asn1.Unmarshal(raw, &field); err != nil {
		return "", fmt.Errorf("parsing login field: %w", err)
	}

	switch field.Algorithm.OID.String() {
	case oidDESEDE3CBC:
		if len(key.bytes) < 24 {
			return "", fmt.Errorf("key too short for 3DES field")
		}
		block, err := des.NewTripleDESCipher(key.bytes[:24])
		if err != nil {
			return "", err
		}
		return decryptFieldBlock(block, field.Algorithm.Params.Bytes, field.Cipher)

	case oidAES256CBC:
		if len(key.bytes) < 32 {
			return "", fmt.Errorf("key too short for AES-256 field")
		}
		block, err := aes.NewCipher(key.bytes[:32])
		if err != nil {
			return "", err
		}
		return decryptFieldBlock(block, field.Algorithm.Params.Bytes, field.Cipher)

	default:
		return "", fmt.Errorf("unsupported field algorithm %s", field.Algorithm.OID)
	}
}

func decryptFieldBlock(block cipher.Block, iv, ciphertext []byte) (string, error) {
	if len(iv) != block.BlockSize() || len(ciphertext)%block.BlockSize() != 0 || len(ciphertext) == 0 {
		return "", fmt.Errorf("malformed field ciphertext")
	}
	out := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, ciphertext)
	plain, ok := stripPKCS7(out, block.BlockSize())
	if !ok {
		return "", fmt.Errorf("invalid padding while decrypting field")
	}
	return string(plain), nil
}

// stripPKCS7 validates and removes PKCS#7 padding for the given block size.
func stripPKCS7(data []byte, blockSize int) ([]byte, bool) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, false
	}
	pad := int(data[len(data)-1])
	if pad < 1 || pad > blockSize || pad > len(data) {
		return nil, false
	}
	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return nil, false
		}
	}
	return data[:len(data)-pad], true
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

package totp

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Code returns the current RFC 6238 code. Secret may be a Base32 key or an
// otpauth:// URI. SHA-1, 30-second periods and six digits are the defaults.
func Code(secret string, at time.Time) (string, error) {
	secret = strings.TrimSpace(secret)
	period, digits := int64(30), 6
	if strings.HasPrefix(strings.ToLower(secret), "otpauth://") {
		parsed, err := url.Parse(secret)
		if err != nil {
			return "", fmt.Errorf("parse otpauth URI: %w", err)
		}
		if !strings.EqualFold(parsed.Host, "totp") {
			return "", fmt.Errorf("only otpauth TOTP entries are supported")
		}
		query := parsed.Query()
		secret = query.Get("secret")
		if value := query.Get("period"); value != "" {
			parsedPeriod, err := strconv.ParseInt(value, 10, 64)
			if err != nil || parsedPeriod <= 0 {
				return "", fmt.Errorf("invalid TOTP period %q", value)
			}
			period = parsedPeriod
		}
		if value := query.Get("digits"); value != "" {
			parsedDigits, err := strconv.Atoi(value)
			if err != nil || parsedDigits < 6 || parsedDigits > 8 {
				return "", fmt.Errorf("invalid TOTP digits %q", value)
			}
			digits = parsedDigits
		}
		if algorithm := query.Get("algorithm"); algorithm != "" && !strings.EqualFold(algorithm, "SHA1") {
			return "", fmt.Errorf("only SHA1 TOTP is supported")
		}
	}
	secret = strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(secret))
	if secret == "" {
		return "", fmt.Errorf("TOTP secret is empty")
	}
	secret += strings.Repeat("=", (8-len(secret)%8)%8)
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("decode Base32 TOTP secret: %w", err)
	}
	counter := uint64(at.Unix() / period)
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], counter)
	hasher := hmac.New(sha1.New, key)
	_, _ = hasher.Write(message[:])
	digest := hasher.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	value := binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7fffffff
	modulus := uint32(1)
	for range digits {
		modulus *= 10
	}
	return fmt.Sprintf("%0*d", digits, value%modulus), nil
}

func Remaining(secret string, at time.Time) int {
	period := int64(30)
	if parsed, err := url.Parse(strings.TrimSpace(secret)); err == nil && strings.EqualFold(parsed.Scheme, "otpauth") {
		if value, err := strconv.ParseInt(parsed.Query().Get("period"), 10, 64); err == nil && value > 0 {
			period = value
		}
	}
	return int(period - at.Unix()%period)
}

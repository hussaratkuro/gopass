// Package generator implements random password generation.
package generator

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	upperChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowerChars  = "abcdefghijklmnopqrstuvwxyz"
	symbolChars = "!@#$%^&*()-_=+[]{}|;:,.<>/?"
	numberChars = "0123456789"
	MinLength   = 1
	MaxLength   = 50
)

// Options controls which character classes a generated password may draw from.
type Options struct {
	Upper   bool
	Lower   bool
	Symbols bool
	Numbers bool
}

// Charset returns the concatenated character set implied by the options.
func (o Options) Charset() string {
	var charset string
	if o.Upper {
		charset += upperChars
	}
	if o.Lower {
		charset += lowerChars
	}
	if o.Symbols {
		charset += symbolChars
	}
	if o.Numbers {
		charset += numberChars
	}
	return charset
}

// Generate produces a random password of the given length using the supplied options.
func Generate(length int, opts Options) (string, error) {
	if length < MinLength || length > MaxLength {
		return "", fmt.Errorf("length must be a number between %d and %d", MinLength, MaxLength)
	}

	charset := opts.Charset()
	if charset == "" {
		return "", fmt.Errorf("select at least one character set")
	}

	password := make([]byte, length)
	for i := range password {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password[i] = charset[index.Int64()]
	}
	return string(password), nil
}

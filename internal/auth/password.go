package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	MinPasswordLength    = 8
	passwordSaltBytes    = 16
	passwordHashBytes    = 32
	passwordArgonTime    = 3
	passwordArgonMemory  = 64 * 1024
	passwordArgonThreads = 2
)

// passwordTooShort is the single length rule for setting a password; every entry point
// (setup, registration, administrative creation, change-password) must route here.
//
// It counts characters (code points), not bytes: len("密码密") is 9, so a byte-based
// bound accepted a three-character password while the browser refused to submit it
// (HTML `minlength` counts UTF-16 code units) and the "at least 8 characters" copy was
// wrong for every non-ASCII password. An HTML code-unit count is never smaller than a
// code-point count, so the client-side bound can only be looser than this one: a short
// non-ASCII password reaches the server and gets the translated error instead of being
// blocked in the browser with a message the UI cannot localise.
func passwordTooShort(password string) bool {
	return utf8.RuneCountInString(strings.TrimSpace(password)) < MinPasswordLength
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: generate password salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, passwordArgonTime, passwordArgonMemory, passwordArgonThreads, passwordHashBytes)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		passwordArgonMemory,
		passwordArgonTime,
		passwordArgonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(password string, encodedHash string) (bool, error) {
	params, salt, hash, err := parsePasswordHash(encodedHash)
	if err != nil {
		return false, err
	}
	hashLength := len(hash)
	if hashLength > math.MaxUint32 {
		return false, errors.New("auth: password hash is too large")
	}
	hashLengthUint32 := uint32(hashLength)

	comparison := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, hashLengthUint32)
	return subtle.ConstantTimeCompare(hash, comparison) == 1, nil
}

type passwordParams struct {
	memory  uint32
	time    uint32
	threads uint8
}

func parsePasswordHash(encodedHash string) (passwordParams, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return passwordParams{}, nil, nil, errors.New("auth: invalid encoded password hash")
	}
	if parts[1] != "argon2id" {
		return passwordParams{}, nil, nil, errors.New("auth: unsupported password hash algorithm")
	}
	if parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return passwordParams{}, nil, nil, errors.New("auth: unsupported password hash version")
	}

	params, err := parsePasswordParams(parts[3])
	if err != nil {
		return passwordParams{}, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return passwordParams{}, nil, nil, fmt.Errorf("auth: invalid password hash salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return passwordParams{}, nil, nil, fmt.Errorf("auth: invalid password hash value: %w", err)
	}

	return params, salt, hash, nil
}

func parsePasswordParams(raw string) (passwordParams, error) {
	var params passwordParams
	for part := range strings.SplitSeq(raw, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return passwordParams{}, errors.New("auth: invalid password hash parameters")
		}

		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return passwordParams{}, fmt.Errorf("auth: invalid password hash parameter %s: %w", key, err)
		}

		switch key {
		case "m":
			params.memory = uint32(parsed)
		case "t":
			params.time = uint32(parsed)
		case "p":
			if parsed > math.MaxUint8 {
				return passwordParams{}, errors.New("auth: invalid password hash threads value")
			}
			params.threads = uint8(parsed)
		default:
			return passwordParams{}, errors.New("auth: unsupported password hash parameter")
		}
	}

	if params.memory == 0 || params.time == 0 || params.threads == 0 {
		return passwordParams{}, errors.New("auth: incomplete password hash parameters")
	}
	return params, nil
}

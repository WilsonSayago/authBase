package secundary

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2MemoryKiB   uint32 = 19 * 1024
	argon2Iterations  uint32 = 2
	argon2Parallelism uint8  = 1
	argon2SaltBytes          = 16
	argon2KeyBytes           = 32

	minArgon2MemoryKiB   uint32 = 8 * 1024
	maxArgon2MemoryKiB   uint32 = 64 * 1024
	maxArgon2Iterations  uint32 = 5
	maxArgon2Parallelism uint8  = 4
	minArgon2SaltBytes          = 16
	maxArgon2SaltBytes          = 32
	minArgon2KeyBytes           = 16
	maxArgon2KeyBytes           = 64

	// MaxPasswordBytes bounds request-path CPU and memory while allowing the
	// application policy to accept at least 128 Unicode runes.
	MaxPasswordBytes = 1024
)

type argon2Parameters struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func hashArgon2id(password string) (string, error) {
	if len(password) > MaxPasswordBytes {
		return "", fmt.Errorf("password exceeds %d bytes", MaxPasswordBytes)
	}
	salt := make([]byte, argon2SaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate argon2id salt: %w", err)
	}
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Iterations,
		argon2MemoryKiB,
		argon2Parallelism,
		argon2KeyBytes,
	)
	encoding := base64.RawStdEncoding
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2MemoryKiB,
		argon2Iterations,
		argon2Parallelism,
		encoding.EncodeToString(salt),
		encoding.EncodeToString(hash),
	), nil
}

func verifyArgon2id(encodedHash, password string) bool {
	if len(password) > MaxPasswordBytes || len(encodedHash) > 512 {
		return false
	}
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false
	}
	if parts[2] != "v="+strconv.Itoa(argon2.Version) {
		return false
	}

	params, ok := parseArgon2Parameters(parts[3])
	if !ok || len(parts[4]) > 64 || len(parts[5]) > 128 {
		return false
	}
	encoding := base64.RawStdEncoding
	salt, err := encoding.DecodeString(parts[4])
	if err != nil || len(salt) < minArgon2SaltBytes || len(salt) > maxArgon2SaltBytes {
		return false
	}
	expected, err := encoding.DecodeString(parts[5])
	if err != nil || len(expected) < minArgon2KeyBytes || len(expected) > maxArgon2KeyBytes {
		return false
	}

	actual := argon2.IDKey(
		[]byte(password),
		salt,
		params.iterations,
		params.memory,
		params.parallelism,
		uint32(len(expected)),
	)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func parseArgon2Parameters(encoded string) (argon2Parameters, bool) {
	parts := strings.Split(encoded, ",")
	if len(parts) != 3 {
		return argon2Parameters{}, false
	}
	memory, ok := parseUintParameter(parts[0], "m=", 32)
	if !ok {
		return argon2Parameters{}, false
	}
	iterations, ok := parseUintParameter(parts[1], "t=", 32)
	if !ok {
		return argon2Parameters{}, false
	}
	parallelism, ok := parseUintParameter(parts[2], "p=", 8)
	if !ok {
		return argon2Parameters{}, false
	}

	params := argon2Parameters{
		memory:      uint32(memory),
		iterations:  uint32(iterations),
		parallelism: uint8(parallelism),
	}
	if params.memory < minArgon2MemoryKiB || params.memory > maxArgon2MemoryKiB {
		return argon2Parameters{}, false
	}
	if params.iterations == 0 || params.iterations > maxArgon2Iterations {
		return argon2Parameters{}, false
	}
	if params.parallelism == 0 || params.parallelism > maxArgon2Parallelism {
		return argon2Parameters{}, false
	}
	return params, true
}

func parseUintParameter(value, prefix string, bits int) (uint64, bool) {
	if !strings.HasPrefix(value, prefix) || len(value) == len(prefix) {
		return 0, false
	}
	parsed, err := strconv.ParseUint(strings.TrimPrefix(value, prefix), 10, bits)
	return parsed, err == nil
}

// Package auth holds the cryptographic primitives behind the login flow:
// password hashing and session token minting. It deliberately knows nothing
// about the database or HTTP — the service layer wires those together — so
// these functions stay testable and reusable on their own.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Current hashing parameters. 64 MiB with three passes is the OWASP-suggested
// argon2id baseline; parallelism 2 keeps a single login well under 100ms on
// the small VM this runs on.
const (
	argonMemory      uint32 = 64 * 1024
	argonTime        uint32 = 3
	argonParallelism uint8  = 2
	argonSaltLen     uint32 = 16
	argonKeyLen      uint32 = 32
)

// HashPassword returns a PHC-formatted argon2id hash:
//
//	$argon2id$v=19$m=65536,t=3,p=2$<salt-b64>$<hash-b64>
//
// The parameters travel inside the string rather than living only in this
// file, which is what makes the constants above safe to raise later.
func HashPassword(plain string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	key := argon2.IDKey([]byte(plain), salt, argonTime, argonMemory, argonParallelism, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword recomputes the hash using the parameters parsed out of the
// stored string, not the constants above. That is the whole point of the PHC
// encoding: raising argonMemory or argonTime tomorrow re-hashes nothing and
// invalidates no existing password, because every old hash still carries the
// settings it was created with.
func VerifyPassword(hash, plain string) (bool, error) {
	parts := strings.Split(hash, "$")
	// Leading empty field from the initial "$", then: id, version, params,
	// salt, key.
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false, fmt.Errorf("password hash is not argon2id PHC format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("parse argon2 version: %w", err)
	}
	if version != argon2.Version {
		return false, fmt.Errorf("unsupported argon2 version %d", version)
	}

	var memory, time uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &parallelism); err != nil {
		return false, fmt.Errorf("parse argon2 params: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("decode argon2 salt: %w", err)
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("decode argon2 hash: %w", err)
	}

	got := argon2.IDKey([]byte(plain), salt, time, memory, parallelism, uint32(len(want)))
	// Constant-time: a byte-by-byte comparison would leak how much of the
	// derived key matched through its timing.
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

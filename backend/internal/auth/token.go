package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// sessionTokenBytes is the entropy behind a session token. 256 bits is far
// past any brute-force concern, and the tokens are never typed by a human so
// their length costs nothing.
const sessionTokenBytes = 32

// NewSessionToken mints a session token and returns it twice: the plaintext
// that goes to the client exactly once, and the SHA-256 digest that is the
// only form the database ever sees. Storing the usable token would make the
// sessions table equivalent to a list of live credentials — anyone who could
// read a backup could impersonate every logged-in user without cracking
// anything.
//
// A plain SHA-256 is the right hash here, unlike for passwords: the input is
// already 256 bits of uniform randomness, so there is nothing for an attacker
// to guess and no reason to pay argon2's cost on every authenticated request.
func NewSessionToken() (plain string, hash []byte, err error) {
	raw := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate session token: %w", err)
	}
	plain = base64.RawURLEncoding.EncodeToString(raw)
	return plain, HashToken(plain), nil
}

// HashToken maps a presented token back to the digest stored in the sessions
// table. It must stay in lockstep with NewSessionToken: it hashes the encoded
// string, not the raw bytes behind it.
func HashToken(plain string) []byte {
	sum := sha256.Sum256([]byte(plain))
	return sum[:]
}

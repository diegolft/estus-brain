package domain

import (
	"strings"
	"time"
)

// User is the authenticated account as the rest of the app sees it. There is
// deliberately no password hash field: the hash never leaves the store and
// the auth service, so no handler can serialize one into a response by
// accident the way it could if the struct carried it around.
type User struct {
	ID        string
	Email     string
	Name      string
	CreatedAt time.Time
}

// NormalizeEmail is the single definition of the form an address is stored
// and looked up in. The users table has a plain unique index rather than a
// citext column, so case-insensitivity only holds as long as every write and
// every lookup passes through here first.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

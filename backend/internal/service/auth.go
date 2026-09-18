package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/rafael/estus-vault/backend/internal/auth"
	"github.com/rafael/estus-vault/backend/internal/domain"
	"github.com/rafael/estus-vault/backend/internal/store/postgres"
)

// SessionTTL is how long a login lasts. This is a single-owner app on a
// personal device, so a month between logins is the right trade: short
// enough that a stolen token eventually dies on its own, long enough that
// the owner is never re-authenticating mid-task.
const SessionTTL = 30 * 24 * time.Hour

// MinPasswordLength is enforced on every path that sets a password. Length
// is the only property worth requiring: composition rules push people toward
// predictable substitutions without adding real entropy.
const MinPasswordLength = 12

// ErrInvalidCredentials is what every failed login returns, whether the
// email was unknown or the password was wrong. Distinguishing the two would
// turn the login form into an account-existence oracle: anyone could probe
// addresses and learn which ones have accounts here.
var ErrInvalidCredentials = errors.New("invalid email or password")

// dummyHash is a real argon2id hash of a throwaway value, used when the
// email doesn't exist. Without it, a missing account would return in the
// microseconds a failed SELECT takes while a real one would pay argon2's
// ~50ms — and that gap alone answers "does this address have an account?"
// for anyone with a stopwatch. Verifying against this constant makes both
// paths cost the same.
const dummyHash = "$argon2id$v=19$m=65536,t=3,p=2$ThR5hdd78KBpd4unLsp0xA$aLVcWoc0hkRqlEcBmck8ZYnAkcs7+wX8AYfZt10KbYM"

type AuthService struct {
	users    *postgres.UserRepo
	sessions *postgres.SessionRepo
}

func NewAuthService(users *postgres.UserRepo, sessions *postgres.SessionRepo) *AuthService {
	return &AuthService{users: users, sessions: sessions}
}

// Login verifies credentials and opens a session. The returned token is the
// only time the usable value exists outside the client: the database gets
// its digest.
func (s *AuthService) Login(ctx context.Context, email, password, userAgent string) (string, time.Time, domain.User, error) {
	user, hash, err := s.users.UserByEmail(ctx, domain.NormalizeEmail(email))
	if errors.Is(err, domain.ErrNotFound) {
		// Burn the same time a real verification costs before giving up.
		_, _ = auth.VerifyPassword(dummyHash, password)
		return "", time.Time{}, domain.User{}, ErrInvalidCredentials
	}
	if err != nil {
		return "", time.Time{}, domain.User{}, err
	}

	ok, err := auth.VerifyPassword(hash, password)
	if err != nil {
		return "", time.Time{}, domain.User{}, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return "", time.Time{}, domain.User{}, ErrInvalidCredentials
	}

	token, tokenHash, err := auth.NewSessionToken()
	if err != nil {
		return "", time.Time{}, domain.User{}, err
	}
	expiresAt := time.Now().Add(SessionTTL)
	if err := s.sessions.CreateSession(ctx, user.ID, tokenHash, expiresAt, userAgent); err != nil {
		return "", time.Time{}, domain.User{}, err
	}
	return token, expiresAt, user, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessions.DeleteSession(ctx, auth.HashToken(token))
}

// UserFromToken is the hot path: it runs on every authenticated request, so
// it does exactly one query and no password hashing.
func (s *AuthService) UserFromToken(ctx context.Context, token string) (domain.User, error) {
	if token == "" {
		return domain.User{}, ErrInvalidCredentials
	}
	user, err := s.sessions.SessionUser(ctx, auth.HashToken(token))
	if errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, ErrInvalidCredentials
	}
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// ChangePassword requires the current password even though the caller is
// already authenticated: it stops someone who walked up to an unlocked
// session from locking the owner out of their own account.
func (s *AuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	ok, err := s.VerifyUserPassword(ctx, userID, currentPassword)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidCredentials
	}
	if len(newPassword) < MinPasswordLength {
		return fmt.Errorf("%w: a senha precisa ter pelo menos %d caracteres", domain.ErrValidation, MinPasswordLength)
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.users.UpdateUserPassword(ctx, userID, hash)
}

// VerifyUserPassword re-confirms the password of an already-authenticated
// user. The vault's reveal gate uses it to make holding a session
// insufficient for reading a stored secret in plaintext.
func (s *AuthService) VerifyUserPassword(ctx context.Context, userID, password string) (bool, error) {
	_, hash, err := s.users.UserByID(ctx, userID)
	if err != nil {
		return false, err
	}
	ok, err := auth.VerifyPassword(hash, password)
	if err != nil {
		return false, fmt.Errorf("verify password: %w", err)
	}
	return ok, nil
}

// EnsureBootstrapUser creates the very first account from the environment so
// a fresh deployment isn't locked out of an app that now requires a login.
// It refuses to do anything once any user exists, which keeps the admin
// variables from silently resurrecting a deleted account or being a standing
// back door on a running installation.
func (s *AuthService) EnsureBootstrapUser(ctx context.Context, email, password string) error {
	if email == "" || password == "" {
		return nil
	}
	n, err := s.users.CountUsers(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if len(password) < MinPasswordLength {
		return fmt.Errorf("%w: ESTUS_ADMIN_PASSWORD precisa ter pelo menos %d caracteres", domain.ErrValidation, MinPasswordLength)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}
	normalized := domain.NormalizeEmail(email)
	if _, err := s.users.CreateUser(ctx, normalized, hash, ""); err != nil {
		return err
	}
	slog.Info("bootstrap user created", "email", normalized)
	return nil
}

// SweepExpiredSessions drops sessions past their expiry. Nothing depends on
// it for correctness — SessionUser already filters on expires_at — it just
// keeps the table from growing without bound.
func (s *AuthService) SweepExpiredSessions(ctx context.Context) (int64, error) {
	return s.sessions.DeleteExpiredSessions(ctx)
}

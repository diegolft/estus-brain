package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/rafael/estus-vault/backend/internal/domain"
)

type SessionRepo struct{ db *DB }

func NewSessionRepo(db *DB) *SessionRepo { return &SessionRepo{db: db} }

// CreateSession takes the token's digest, never the token itself — see
// auth.NewSessionToken for why the usable value stays out of the database.
func (r *SessionRepo) CreateSession(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time, userAgent string) error {
	_, err := r.db.Pool.Exec(ctx, `
		insert into sessions (id, user_id, token_hash, expires_at, user_agent)
		values (gen_random_uuid(), $1, $2, $3, $4)`,
		userID, tokenHash, expiresAt, userAgent)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// SessionUser resolves a token digest to its owner in one round trip. The
// expiry is filtered in SQL rather than compared in Go so that an expired
// session is indistinguishable from a forged one to the caller, and so the
// clock that decides is the database's — the same one that wrote expires_at.
func (r *SessionRepo) SessionUser(ctx context.Context, tokenHash []byte) (domain.User, error) {
	var u domain.User
	err := r.db.Pool.QueryRow(ctx, `
		select u.id, u.email, u.name, u.created_at
		from sessions s
		join users u on u.id = s.user_id
		where s.token_hash = $1 and s.expires_at > now()`, tokenHash,
	).Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get session user: %w", err)
	}
	return u, nil
}

// DeleteSession is logout. A token that was already absent is not an error:
// logging out twice, or after the sweep already collected the row, should
// still leave the caller logged out rather than fail.
func (r *SessionRepo) DeleteSession(ctx context.Context, tokenHash []byte) error {
	if _, err := r.db.Pool.Exec(ctx, `delete from sessions where token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (r *SessionRepo) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	tag, err := r.db.Pool.Exec(ctx, `delete from sessions where expires_at <= now()`)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}

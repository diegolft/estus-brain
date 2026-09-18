package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/rafael/estus-vault/backend/internal/domain"
)

type UserRepo struct{ db *DB }

func NewUserRepo(db *DB) *UserRepo { return &UserRepo{db: db} }

// CreateUser expects an already-normalized email; see domain.NormalizeEmail.
// The caller passing a raw address would break the unique index's promise of
// case-insensitivity rather than fail loudly, so the auth service normalizes
// before it ever gets here.
func (r *UserRepo) CreateUser(ctx context.Context, email, passwordHash, name string) (domain.User, error) {
	var u domain.User
	err := r.db.Pool.QueryRow(ctx, `
		insert into users (id, email, password_hash, name)
		values (gen_random_uuid(), $1, $2, $3)
		returning id, email, name, created_at`,
		email, passwordHash, name,
	).Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// UserByEmail returns the account plus its password hash. The hash is a
// second return value instead of a struct field so it can't ride along into
// a response body — domain.User has nowhere to put it.
func (r *UserRepo) UserByEmail(ctx context.Context, email string) (domain.User, string, error) {
	var u domain.User
	var hash string
	err := r.db.Pool.QueryRow(ctx, `
		select id, email, name, created_at, password_hash
		from users where email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, "", domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, "", fmt.Errorf("get user by email: %w", err)
	}
	return u, hash, nil
}

func (r *UserRepo) UserByID(ctx context.Context, id string) (domain.User, string, error) {
	var u domain.User
	var hash string
	err := r.db.Pool.QueryRow(ctx, `
		select id, email, name, created_at, password_hash
		from users where id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, "", domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, "", fmt.Errorf("get user by id: %w", err)
	}
	return u, hash, nil
}

// CountUsers backs the bootstrap check on boot: a zero count is what makes
// creating the first account from the environment safe.
func (r *UserRepo) CountUsers(ctx context.Context) (int, error) {
	var n int
	if err := r.db.Pool.QueryRow(ctx, `select count(*) from users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

func (r *UserRepo) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	tag, err := r.db.Pool.Exec(ctx, `
		update users set password_hash = $1, updated_at = now() where id = $2`,
		passwordHash, id)
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

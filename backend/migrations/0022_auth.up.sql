-- Email/password login replaces the mTLS edge that used to be the only gate
-- in front of this app. The edge certificate protected the whole deployment
-- indiscriminately; a real session lets the API tell one request from
-- another and lets the vault re-confirm identity before revealing a secret.

-- Email is treated as case-insensitive, but the column is plain text with a
-- plain unique index: citext would be the natural fit and this database has
-- never installed that extension, so adding it here would make the migration
-- fail on any deployment where the role can't CREATE EXTENSION. The
-- application lowercases every address before it reaches SQL instead, which
-- keeps the uniqueness guarantee on the same normalized form.
create table users (
    id            uuid primary key default gen_random_uuid(),
    email         text not null unique,
    password_hash text not null,
    name          text not null default '',
    created_at    timestamptz not null default now(),
    updated_at    timestamptz not null default now()
);

-- Only the SHA-256 of a session token is stored. A leaked database dump
-- therefore contains nothing that can be replayed as a cookie or bearer
-- token; the usable value exists only in the client's hands.
create table sessions (
    id         uuid primary key default gen_random_uuid(),
    user_id    uuid not null references users(id) on delete cascade,
    token_hash bytea not null unique,
    expires_at timestamptz not null,
    created_at timestamptz not null default now(),
    user_agent text not null default ''
);

-- Deleting a user cascades through user_id, and the expiry sweep scans by
-- expires_at; both run often enough to be worth an index.
create index sessions_user_idx on sessions (user_id);
create index sessions_expires_idx on sessions (expires_at);

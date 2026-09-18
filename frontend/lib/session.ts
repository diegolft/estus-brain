import "server-only";
import { cookies } from "next/headers";

// Everything about the browser's session lives here, and only here, so there
// is one place to reason about when a token is written, read or thrown away.
// Like lib/api.ts this module is server-only: the token must never reach a
// client bundle.
//
// The browser never talks to the Go API — it talks to Next.js, and Next.js
// talks to Go. So the session cookie is a cookie of *this* app, and the Go
// API's own credential (the bearer token) is only ever attached on the
// server, in authHeaders() below.
export const API_URL = process.env.API_URL ?? "http://localhost:8080";

export const SESSION_COOKIE = "estus_session";

export interface SessionUser {
  id: string;
  email: string;
  name: string;
}

// httpOnly is the whole point of this design: the token is a bearer
// credential for the Go API, so a single XSS anywhere in the app would be
// enough to read it out of document.cookie and replay it from anywhere. With
// httpOnly the browser will hand it back to Next.js on every request and
// still refuse to show it to any script running on the page. sameSite "lax"
// keeps it off cross-site POSTs while letting an ordinary link into the app
// still arrive logged in, and secure is on wherever there's real TLS —
// development runs over plain http, where a secure cookie would simply never
// be stored.
function cookieOptions(maxAge: number) {
  return {
    httpOnly: true,
    sameSite: "lax" as const,
    path: "/",
    secure: process.env.NODE_ENV === "production",
    maxAge,
  };
}

export async function getSessionToken(): Promise<string | null> {
  // cookies() is async in Next 16 — it's a request-time API, so reading it
  // opts the caller into dynamic rendering.
  const store = await cookies();
  return store.get(SESSION_COOKIE)?.value ?? null;
}

// expiresAt is whatever the Go API said when it minted the token, so the
// cookie dies exactly when the session does rather than on a lifetime this
// side invented. A token that somehow arrives already expired gets a maxAge
// of zero, which tells the browser to drop it immediately.
export async function setSessionCookie(token: string, expiresAt: string): Promise<void> {
  const seconds = Math.floor((new Date(expiresAt).getTime() - Date.now()) / 1000);
  const maxAge = Number.isFinite(seconds) && seconds > 0 ? seconds : 0;
  const store = await cookies();
  store.set(SESSION_COOKIE, token, cookieOptions(maxAge));
}

export async function clearSessionCookie(): Promise<void> {
  const store = await cookies();
  store.delete(SESSION_COOKIE);
}

// Returns an empty object rather than throwing when there's no session: the
// Go API answers 401 on its own, and letting the request go out unauthorized
// keeps the failure in one place instead of two.
export async function authHeaders(): Promise<Record<string, string>> {
  const token = await getSessionToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

// Resolves to null for anyone who isn't logged in — including a cookie the Go
// API no longer recognises, which is the case the proxy deliberately can't
// catch. Pages that want the viewer's name can call this; the redirect to the
// login screen is the proxy's job.
export async function getCurrentUser(): Promise<SessionUser | null> {
  const headers = await authHeaders();
  if (!("Authorization" in headers)) return null;

  const res = await fetch(`${API_URL}/api/auth/me`, { headers, cache: "no-store" });
  if (!res.ok) return null;
  return res.json() as Promise<SessionUser>;
}

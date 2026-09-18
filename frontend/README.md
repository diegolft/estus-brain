# Estus Brain — frontend

The Next.js app. It is the only thing the browser talks to: the Go API has no
public listener, so every read and write goes through a server component,
server action or route handler here, which is what makes the session design
below possible.

## Running

```bash
npm install
npm run dev     # http://localhost:3000
npm run build
```

## Environment

Copy `.env.example` to `.env.local` and adjust.

| Variable  | Default                 | What it is                                                                                                                                                                           |
| --------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `API_URL` | `http://localhost:8080` | Base URL of the Go API, read **server-side only**. Deliberately not `NEXT_PUBLIC_`: the address must never reach the browser bundle, and `lib/api.ts` imports `server-only` to enforce it. |

## Sessions

Signing in posts to the Go API's `POST /api/auth/login` from a server action.
The token that comes back is stored in an **httpOnly** cookie named
`estus_session` — httpOnly because the token is a bearer credential, and a
cookie readable from `document.cookie` would turn any XSS anywhere in the app
into a stolen session. No client-side code ever sees it.

- `lib/session.ts` owns the cookie: reading, writing, clearing, and turning it
  into the `Authorization: Bearer …` header.
- `lib/serverFetch.ts` is the single fetch wrapper every `lib/` module and
  route handler uses, so no call can reach Go without that header.
- `proxy.ts` (Next 16's rename of `middleware.ts`) redirects anyone without
  the cookie to `/login`, carrying where they were headed in `?next`. It only
  checks that the cookie *exists* — validating it is the Go API's job, and
  proxy code runs on every request, so it does no network I/O.

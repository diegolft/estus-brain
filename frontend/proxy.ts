import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

// Next 16 renamed the middleware file convention to proxy; the behaviour is
// the same, the name is not (see node_modules/next/dist/docs — middleware.md
// is now only a deprecation notice pointing here).
//
// This is an optimistic check and nothing more. It asks whether the request
// carries a session cookie at all, never whether that cookie is still good:
// the only authority on that is the Go API, and proxy code runs on every
// single request, so calling out to it here would put a network round trip
// in front of every page, image and navigation in the app. A revoked or
// expired token gets past this and is rejected the moment the page's own
// server-side fetch reaches Go with a 401.
const SESSION_COOKIE = "estus_session";

export function proxy(request: NextRequest) {
  if (request.cookies.has(SESSION_COOKIE)) return NextResponse.next();

  // Where they were trying to go travels in ?next so the login screen can
  // hand them back to it instead of dumping everyone at the núcleo. Only the
  // path and query ride along — never an absolute URL, which would let a
  // crafted link bounce someone off this app onto another site after login.
  const url = new URL("/login", request.url);
  const wanted = `${request.nextUrl.pathname}${request.nextUrl.search}`;
  if (wanted !== "/") url.searchParams.set("next", wanted);
  return NextResponse.redirect(url);
}

export const config = {
  // Everything except the login screen itself, the MCP endpoint, Next's own
  // build output, the favicon, and anything with a file extension. /mcp is
  // reached by AI apps from outside the browser carrying their own bearer
  // token and no cookie at all — bouncing them to a login page would break
  // them. The fonts and images under public/ are fetched like any other
  // request, and guarding those would keep the login page from ever
  // rendering its own assets.
  matcher: ["/((?!login|mcp|_next/static|_next/image|favicon.ico|.*\\.[^/]+$).*)"],
};

import "server-only";
import { API_URL, authHeaders } from "./session";

// Every module under lib/ used to carry its own copy of the same fetch
// wrapper. Now that each of those calls has to arrive at the Go API carrying
// the viewer's bearer token, one forgotten copy would be a silent 401 on one
// screen, so the wrapper lives here and the modules keep only what is
// actually theirs: the DTOs and the paths.

export { API_URL };

/**
 * The raw call: attaches the session's Authorization header and hands back the
 * Response untouched. For callers that read the status themselves (a 404 that
 * means "not found" rather than an error) or that stream a body through.
 */
export async function goFetch(path: string, init?: RequestInit): Promise<Response> {
  return fetch(`${API_URL}${path}`, {
    ...init,
    headers: { ...(await authHeaders()), ...init?.headers },
  });
}

/**
 * The JSON call: sends and expects JSON, throws on any non-2xx, and resolves
 * to undefined for a 204. A DELETE that removed the resource replies 204 with
 * an empty body — the correct response, cemented by a backend test — so
 * res.json() would throw "Unexpected end of JSON input" on an empty string.
 */
export async function goJson<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await goFetch(path, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`estus-vault api ${path} -> ${res.status}: ${body}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

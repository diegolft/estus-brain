"use server";

import { redirect } from "next/navigation";
import { API_URL, authHeaders, clearSessionCookie, setSessionCookie } from "@/lib/session";

export type LoginState = { error?: string };

// One message for every way a sign-in can fail. Telling someone the e-mail
// exists but the password is wrong turns the login form into a way to find
// out who has an account here, so both cases — and a backend that is simply
// down — read the same.
const GENERIC_FAILURE = "E-mail ou senha incorretos.";

interface LoginResponse {
  token: string;
  expires_at: string;
  user: { id: string; email: string; name: string };
}

export async function loginAction(_prev: LoginState, formData: FormData): Promise<LoginState> {
  const email = String(formData.get("email") ?? "").trim();
  const password = String(formData.get("password") ?? "");
  // Only a path is ever accepted here: an absolute URL in ?next would make
  // this form an open redirect, bouncing someone off to another site with
  // the trust of having just logged in.
  const rawNext = String(formData.get("next") ?? "");
  const next = rawNext.startsWith("/") && !rawNext.startsWith("//") ? rawNext : "/";

  if (!email || !password) return { error: GENERIC_FAILURE };

  let data: LoginResponse;
  try {
    const res = await fetch(`${API_URL}/api/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
      cache: "no-store",
    });
    if (!res.ok) return { error: GENERIC_FAILURE };
    data = (await res.json()) as LoginResponse;
  } catch {
    return { error: GENERIC_FAILURE };
  }

  await setSessionCookie(data.token, data.expires_at);
  // redirect() works by throwing, so it has to be outside the try above —
  // inside, the catch would swallow it and report a failed login instead.
  redirect(next);
}

export async function logoutAction(): Promise<void> {
  // The cookie goes either way. If the API can't be reached the token stays
  // alive on its side until it expires, but leaving the browser holding a
  // session it believes in would be worse.
  try {
    await fetch(`${API_URL}/api/auth/logout`, {
      method: "POST",
      headers: await authHeaders(),
      cache: "no-store",
    });
  } catch {
    // Nothing useful to do: the sign-out below still happens.
  }
  await clearSessionCookie();
  redirect("/login");
}

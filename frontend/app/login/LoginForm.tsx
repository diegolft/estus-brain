"use client";

import { useActionState } from "react";
import { loginAction, type LoginState } from "./actions";

const initialState: LoginState = {};

// A client component only because the form needs somewhere to show what came
// back. The credentials themselves are posted by the server action — they are
// never handed to fetch in the browser, so the password never travels through
// client-side JavaScript that an extension or an injected script could read.
export function LoginForm({ next }: { next: string }) {
  const [state, formAction, pending] = useActionState<LoginState, FormData>(loginAction, initialState);

  return (
    <form action={formAction} className="form-grid">
      <input type="hidden" name="next" value={next} />

      <div className="field">
        <label htmlFor="login-email">E-mail</label>
        <input
          id="login-email"
          name="email"
          type="email"
          autoComplete="username"
          placeholder="voce@email.com"
          required
          autoFocus
        />
      </div>

      <div className="field">
        <label htmlFor="login-password">Senha</label>
        <input
          id="login-password"
          name="password"
          type="password"
          autoComplete="current-password"
          placeholder="Sua senha"
          required
        />
      </div>

      <button className="btn-block" type="submit" disabled={pending}>
        {pending ? "Entrando…" : "Entrar"}
      </button>

      {state.error && (
        <p className="form-error" role="alert">
          {state.error}
        </p>
      )}
    </form>
  );
}

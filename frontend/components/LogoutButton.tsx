"use client";

import { useTransition } from "react";
import { logoutAction } from "@/app/login/actions";
import { IconLock } from "./icons";

// Signing out is a mutation, so it goes through a server action rather than a
// link: the action is what can reach the Go API with the token and drop the
// httpOnly cookie, neither of which this side can touch.
export function LogoutButton({ label = false }: { label?: boolean }) {
  const [pending, startTransition] = useTransition();

  return (
    <button
      type="button"
      className={label ? "btn-text" : "icon-btn"}
      aria-label="Sair"
      title="Sair"
      disabled={pending}
      onClick={() => startTransition(() => logoutAction())}
    >
      <IconLock />
      {label && <span>{pending ? "Saindo…" : "Sair"}</span>}
    </button>
  );
}

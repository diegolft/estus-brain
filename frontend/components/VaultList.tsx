"use client";

import { useEffect, useRef, useState } from "react";
import { deleteVaultEntryAction } from "@/app/senhas/actions";
import { VaultEntryForm } from "@/components/VaultEntryForm";
import { Modal } from "@/components/Modal";
import type { VaultEntry } from "@/lib/vault";
import { IconSearch } from "./icons";

const REVEAL_SECONDS = 10;

type RevealState = { password: string; secondsLeft: number };

export function VaultList({ entries }: { entries: VaultEntry[] }) {
  const [revealed, setRevealed] = useState<Record<string, RevealState>>({});
  const [revealing, setRevealing] = useState<string | null>(null);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [editingId, setEditingId] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  // Which entry is waiting on the owner to prove who they are. Being logged
  // in is not enough to read a stored password: the backend asks for the
  // account password again on every reveal, so an unattended open session
  // can't be walked up to and emptied.
  const [confirmingId, setConfirmingId] = useState<string | null>(null);
  const [confirmPassword, setConfirmPassword] = useState("");
  const [confirmError, setConfirmError] = useState("");
  const timers = useRef<Record<string, ReturnType<typeof setInterval>>>({});

  useEffect(() => {
    const activeTimers = timers.current;
    return () => {
      Object.values(activeTimers).forEach(clearInterval);
    };
  }, []);

  function startCountdown(id: string) {
    clearInterval(timers.current[id]);
    timers.current[id] = setInterval(() => {
      setRevealed((prev) => {
        const current = prev[id];
        if (!current) return prev;
        if (current.secondsLeft <= 1) {
          const next = { ...prev };
          delete next[id];
          clearInterval(timers.current[id]);
          return next;
        }
        return { ...prev, [id]: { ...current, secondsLeft: current.secondsLeft - 1 } };
      });
    }, 1000);
  }

  function closeConfirm() {
    setConfirmingId(null);
    setConfirmPassword("");
    setConfirmError("");
  }

  async function revelar(id: string, password: string) {
    setErrors((e) => ({ ...e, [id]: "" }));
    setConfirmError("");
    setRevealing(id);
    try {
      const res = await fetch(`/api/vault/${id}/reveal`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ password }),
      });
      // 401 here is the re-confirmation failing, not the session: the modal
      // stays open so the owner can try again without losing their place.
      if (res.status === 401 || res.status === 403) {
        setConfirmError("Senha incorreta.");
        return;
      }
      if (!res.ok) throw new Error("Não foi possível revelar a senha.");
      const { password: secret } = await res.json();

      setRevealed((prev) => ({ ...prev, [id]: { password: secret, secondsLeft: REVEAL_SECONDS } }));
      startCountdown(id);
      closeConfirm();
    } catch (err) {
      setConfirmError(friendlyMessage(err));
    } finally {
      setRevealing(null);
    }
  }

  async function copiar(password: string) {
    try {
      await navigator.clipboard.writeText(password);
    } catch {
      // Clipboard access can be denied by the browser; the password is
      // still visible on screen so the user can copy it by hand.
    }
  }

  if (entries.length === 0) {
    return <p className="empty-note">Nenhuma senha cadastrada ainda.</p>;
  }

  const q = query.trim().toLowerCase();
  const visibleEntries = q
    ? entries.filter((e) => e.title.toLowerCase().includes(q) || e.username.toLowerCase().includes(q))
    : entries;

  const confirmingEntry = entries.find((e) => e.id === confirmingId) ?? null;

  return (
    <div className="vault-list">
      <div className="list-filters">
        <div className="filter-search">
          <IconSearch />
          <input type="text" placeholder="Buscar senhas" value={query} onChange={(e) => setQuery(e.target.value)} />
        </div>
      </div>
      {visibleEntries.length === 0 && <p className="empty-note">Nenhuma senha bate com essa busca.</p>}
      {visibleEntries.map((entry) => {
        const state = revealed[entry.id];
        return (
          <div className="vault-row" key={entry.id}>
            {editingId === entry.id ? (
              <div className="vault-row-edit">
                <VaultEntryForm entry={entry} onDone={() => setEditingId(null)} />
                <button className="btn-text" type="button" onClick={() => setEditingId(null)}>
                  Cancelar
                </button>
              </div>
            ) : (
              <>
                <div className="vault-row-main">
                  <div className="vault-row-title">{entry.title}</div>
                  <div className="vault-row-meta">{entry.username || "sem usuário"}</div>
                </div>

                <div className="vault-row-password">
                  {state ? (
                    <span className="vault-password-reveal">
                      <code>{state.password}</code>
                      <span className="vault-countdown">{state.secondsLeft}s</span>
                    </span>
                  ) : (
                    <span className="vault-password-mask">••••••••••</span>
                  )}
                </div>

                <div className="vault-row-actions">
                  {state ? (
                    <button className="btn-text" type="button" onClick={() => copiar(state.password)}>
                      Copiar
                    </button>
                  ) : (
                    <button
                      className="btn-text"
                      type="button"
                      onClick={() => {
                        setConfirmPassword("");
                        setConfirmError("");
                        setConfirmingId(entry.id);
                      }}
                      disabled={revealing === entry.id}
                    >
                      {revealing === entry.id ? "Revelando…" : "Revelar"}
                    </button>
                  )}
                  <button className="btn-text" type="button" onClick={() => setEditingId(entry.id)}>
                    Editar
                  </button>
                  <form action={deleteVaultEntryAction.bind(null, entry.id)}>
                    <button className="btn-text bad" type="submit">
                      Excluir
                    </button>
                  </form>
                </div>

                {errors[entry.id] && <p className="form-error vault-row-error">{errors[entry.id]}</p>}
              </>
            )}
          </div>
        );
      })}

      <Modal open={confirmingEntry !== null} onClose={closeConfirm}>
        <div className="panel">
          <div className="panel-head">
            <h2>Confirme sua senha</h2>
          </div>
          <form
            className="form-grid"
            onSubmit={(e) => {
              e.preventDefault();
              if (confirmingEntry) revelar(confirmingEntry.id, confirmPassword);
            }}
          >
            <p className="empty-note">
              Para ver a senha de <b>{confirmingEntry?.title}</b>, digite a senha da sua conta.
            </p>
            <div className="field">
              <label htmlFor="vault-confirm">Senha da conta</label>
              <input
                id="vault-confirm"
                type="password"
                autoComplete="current-password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
                autoFocus
              />
            </div>
            <button className="btn-block" type="submit" disabled={revealing !== null || !confirmPassword}>
              {revealing !== null ? "Revelando…" : "Revelar senha"}
            </button>
            {confirmError && <p className="form-error">{confirmError}</p>}
          </form>
        </div>
      </Modal>
    </div>
  );
}

function friendlyMessage(err: unknown): string {
  if (err instanceof DOMException && err.name === "NotAllowedError") {
    return "Confirmação cancelada.";
  }
  if (err instanceof Error) return err.message;
  return "Não foi possível revelar a senha.";
}

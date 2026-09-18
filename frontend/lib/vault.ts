import "server-only";
import { goJson } from "./serverFetch";

// Mirrors backend/internal/httpapi/dto_vault.go by hand, same convention as
// lib/api.ts and lib/reminders.ts. Reveal isn't here — it runs client-side
// against app/api/vault/[id]/reveal/route.ts, not through this server-only
// module.

export interface VaultEntry {
  id: string;
  title: string;
  username: string;
  url: string;
  notes?: string;
  updated_at: string;
}

export interface VaultEntryInput {
  title: string;
  username: string;
  password: string;
  url: string;
  notes: string;
}

export function listVaultEntries(): Promise<VaultEntry[]> {
  return goJson<VaultEntry[]>("/api/vault", { cache: "no-store" });
}

export function createVaultEntry(input: VaultEntryInput): Promise<VaultEntry> {
  return goJson<VaultEntry>("/api/vault", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateVaultEntry(id: string, input: VaultEntryInput): Promise<VaultEntry> {
  return goJson<VaultEntry>(`/api/vault/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteVaultEntry(id: string): Promise<void> {
  return goJson<void>(`/api/vault/${id}`, { method: "DELETE" });
}

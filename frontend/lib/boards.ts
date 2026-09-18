import "server-only";
import { goFetch, goJson } from "./serverFetch";

// Mirrors backend/internal/httpapi/handlers_boards.go by hand, same convention
// as the other modules. Scenes are saved by the editor through the route
// handler under app/api/boards — they're too big for a server action.

export interface BoardSummary {
  id: string;
  name: string;
  preview: string;
  created_at: string;
  updated_at: string;
}

export interface Board extends BoardSummary {
  scene: Record<string, unknown>;
}

export function listBoards(): Promise<BoardSummary[]> {
  return goJson<BoardSummary[]>("/api/boards", { cache: "no-store" });
}

// Resolves to null for a board that doesn't exist, so the page can 404.
export async function getBoard(id: string): Promise<Board | null> {
  const res = await goFetch(`/api/boards/${encodeURIComponent(id)}`, { cache: "no-store" });
  if (res.status === 404 || res.status === 422) return null;
  if (!res.ok) throw new Error(`estus-vault api /api/boards/${id} -> ${res.status}: ${await res.text()}`);
  return res.json() as Promise<Board>;
}

export function createBoard(name: string): Promise<BoardSummary> {
  return goJson<BoardSummary>("/api/boards", { method: "POST", body: JSON.stringify({ name }) });
}

export function renameBoard(id: string, name: string): Promise<void> {
  return goJson<void>(`/api/boards/${id}`, { method: "PATCH", body: JSON.stringify({ name }) });
}

export function deleteBoard(id: string): Promise<void> {
  return goJson<void>(`/api/boards/${id}`, { method: "DELETE" });
}

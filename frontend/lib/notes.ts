import "server-only";
import { goFetch, goJson } from "./serverFetch";

// Mirrors the JSON DTOs in backend/internal/httpapi/dto_notes.go by hand,
// same convention as the other modules. The editor saves through the route
// handler under app/api/notes — a note with images is past the server-action
// body limit.
export interface Note {
  id: string;
  title: string;
  /** Plain text of the note, kept alongside the document for search and previews. */
  body: string;
  /** The editor's document. Absent in the list, and on notes from before the editor. */
  content?: Record<string, unknown>;
  pinned: boolean;
  category_id?: string;
  created_at: string;
  updated_at: string;
}

export interface NoteCategory {
  id: string;
  name: string;
  color: string;
  created_at: string;
}

export interface NoteInput {
  title: string;
  body: string;
  content?: Record<string, unknown> | null;
  pinned?: boolean;
  category_id?: string | null;
}

export interface NoteCategoryInput {
  name: string;
  color: string;
}

export function listNotes(): Promise<Note[]> {
  return goJson<Note[]>("/api/notes", { cache: "no-store" });
}

// Resolves to null for a note that doesn't exist, so the page can move on.
export async function getNote(id: string): Promise<Note | null> {
  const res = await goFetch(`/api/notes/${encodeURIComponent(id)}`, { cache: "no-store" });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`estus-vault api /api/notes/${id} -> ${res.status}: ${await res.text()}`);
  return res.json() as Promise<Note>;
}

export function createNote(input: NoteInput): Promise<Note> {
  return goJson<Note>("/api/notes", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateNote(id: string, input: NoteInput): Promise<Note> {
  return goJson<Note>(`/api/notes/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteNote(id: string): Promise<void> {
  return goJson<void>(`/api/notes/${id}`, { method: "DELETE" });
}

export function listNoteCategories(): Promise<NoteCategory[]> {
  return goJson<NoteCategory[]>("/api/note-categories", { cache: "no-store" });
}

export function createNoteCategory(input: NoteCategoryInput): Promise<NoteCategory> {
  return goJson<NoteCategory>("/api/note-categories", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateNoteCategory(id: string, input: NoteCategoryInput): Promise<NoteCategory> {
  return goJson<NoteCategory>(`/api/note-categories/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteNoteCategory(id: string): Promise<unknown> {
  return goJson(`/api/note-categories/${id}`, { method: "DELETE" });
}

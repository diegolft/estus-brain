import "server-only";
import { goJson } from "./serverFetch";

// Mirrors backend/internal/httpapi/dto_events.go by hand, same convention as
// lib/reminders.ts and lib/bills.ts — kept as its own file so this module
// never needs to touch the shared types.ts / api.ts other work is editing
// in parallel.

export interface Event {
  id: string;
  title: string;
  location: string;
  notes: string;
  starts_at: string;
  ends_at: string;
  google_event_id?: string;
}

export interface CreateEventInput {
  title: string;
  location?: string;
  notes?: string;
  starts_at: string;
  ends_at: string;
}

export interface GoogleStatus {
  connected: boolean;
}

export interface GoogleSyncResult {
  imported: number;
}

export function listEvents(from: Date, to: Date): Promise<Event[]> {
  const query = `?from=${encodeURIComponent(from.toISOString())}&to=${encodeURIComponent(to.toISOString())}`;
  return goJson<Event[]>(`/api/events${query}`, { cache: "no-store" });
}

export function createEvent(input: CreateEventInput): Promise<Event> {
  return goJson<Event>("/api/events", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateEvent(id: string, input: CreateEventInput): Promise<Event> {
  return goJson<Event>(`/api/events/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteEvent(id: string): Promise<void> {
  return goJson<void>(`/api/events/${id}`, { method: "DELETE" });
}

export function googleStatus(): Promise<GoogleStatus> {
  return goJson<GoogleStatus>("/api/google/status", { cache: "no-store" });
}

export function triggerGoogleSync(): Promise<GoogleSyncResult> {
  return goJson<GoogleSyncResult>("/api/google/sync", { method: "POST" });
}

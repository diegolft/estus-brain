import "server-only";
import { goJson } from "./serverFetch";

// Mirrors backend/internal/httpapi/dto_reminders.go by hand, same convention
// as lib/api.ts and lib/types.ts.

export interface Reminder {
  id: string;
  title: string;
  due_at?: string;
  done: boolean;
  created_at: string;
}

export interface CreateReminderInput {
  title: string;
  due_at?: string;
}

export function listReminders(): Promise<Reminder[]> {
  return goJson<Reminder[]>("/api/reminders", { cache: "no-store" });
}

export function createReminder(input: CreateReminderInput): Promise<Reminder> {
  return goJson<Reminder>("/api/reminders", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function setReminderDone(id: string, done: boolean): Promise<Reminder> {
  return goJson<Reminder>(`/api/reminders/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ done }),
  });
}

export function deleteReminder(id: string): Promise<void> {
  return goJson<void>(`/api/reminders/${id}`, { method: "DELETE" });
}

export function updateReminder(id: string, input: CreateReminderInput): Promise<Reminder> {
  return goJson<Reminder>(`/api/reminders/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

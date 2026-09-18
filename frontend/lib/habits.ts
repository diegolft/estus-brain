import "server-only";
import { goJson } from "./serverFetch";

// Mirrors backend/internal/httpapi/handlers_habits.go by hand. Days are
// "YYYY-MM-DD" in the owner's time zone, worked out here — the API has none.

export type HabitKind = "check" | "count";

export interface Habit {
  id: string;
  name: string;
  kind: HabitKind;
  target: number;
  unit: string;
  days: number[];
  color: string;
  archived: boolean;
  start_day: string;
  current_streak: number;
  best_streak: number;
  /** Day → how much was done, for the days asked for. */
  logs: Record<string, number>;
  created_at: string;
}

export interface HabitInput {
  name: string;
  kind: HabitKind;
  target: number;
  unit: string;
  days: number[];
  color: string;
  archived?: boolean;
  start_day?: string;
}

export function listHabits(today: string, days = 7): Promise<Habit[]> {
  return goJson<Habit[]>(`/api/habits?today=${today}&days=${days}`, { cache: "no-store" });
}

export function createHabit(input: HabitInput): Promise<Habit> {
  return goJson<Habit>("/api/habits", { method: "POST", body: JSON.stringify(input) });
}

export function updateHabit(id: string, input: HabitInput): Promise<void> {
  return goJson<void>(`/api/habits/${id}`, { method: "PUT", body: JSON.stringify(input) });
}

export function deleteHabit(id: string): Promise<void> {
  return goJson<void>(`/api/habits/${id}`, { method: "DELETE" });
}

export function setHabitLog(id: string, day: string, count: number): Promise<void> {
  return goJson<void>(`/api/habits/${id}/log`, { method: "PUT", body: JSON.stringify({ day, count }) });
}

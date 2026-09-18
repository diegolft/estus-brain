import "server-only";
import { goJson } from "./serverFetch";

// Mirrors backend/internal/httpapi/dto_training_diet.go by hand, same
// convention as the other modules.

export interface Exercise {
  id?: string;
  name: string;
  sets: number;
  reps: string;
  weight: string;
  rest_seconds: number;
  notes: string;
}

export interface Workout {
  id: string;
  name: string;
  focus: string;
  /** 0 = Sunday … 6 = Saturday */
  days: number[];
  notes: string;
  exercises: Exercise[];
  created_at: string;
  updated_at: string;
}

export type WorkoutInput = Pick<Workout, "name" | "focus" | "days" | "notes" | "exercises">;

export function listWorkouts(): Promise<Workout[]> {
  return goJson<Workout[]>("/api/workouts", { cache: "no-store" });
}

export function createWorkout(input: WorkoutInput): Promise<Workout> {
  return goJson<Workout>("/api/workouts", { method: "POST", body: JSON.stringify(input) });
}

export function updateWorkout(id: string, input: WorkoutInput): Promise<Workout> {
  return goJson<Workout>(`/api/workouts/${id}`, { method: "PUT", body: JSON.stringify(input) });
}

export function deleteWorkout(id: string): Promise<void> {
  return goJson<void>(`/api/workouts/${id}`, { method: "DELETE" });
}

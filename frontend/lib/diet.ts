import "server-only";
import { goJson } from "./serverFetch";

// Mirrors backend/internal/httpapi/dto_training_diet.go by hand, same
// convention as the other modules.

export interface MealItem {
  id?: string;
  food: string;
  quantity: string;
  kcal: number;
  protein_g: number;
  carbs_g: number;
  fat_g: number;
}

export interface Meal {
  id: string;
  name: string;
  /** "HH:MM", local time */
  time: string;
  /** 0 = Sunday … 6 = Saturday */
  days: number[];
  notes: string;
  items: MealItem[];
  created_at: string;
  updated_at: string;
}

export type MealInput = Pick<Meal, "name" | "time" | "days" | "notes" | "items">;

export interface DietTargets {
  kcal: number;
  protein_g: number;
  carbs_g: number;
  fat_g: number;
}

export function listMeals(): Promise<Meal[]> {
  return goJson<Meal[]>("/api/meals", { cache: "no-store" });
}

export function createMeal(input: MealInput): Promise<Meal> {
  return goJson<Meal>("/api/meals", { method: "POST", body: JSON.stringify(input) });
}

export function updateMeal(id: string, input: MealInput): Promise<Meal> {
  return goJson<Meal>(`/api/meals/${id}`, { method: "PUT", body: JSON.stringify(input) });
}

export function deleteMeal(id: string): Promise<void> {
  return goJson<void>(`/api/meals/${id}`, { method: "DELETE" });
}

export function getDietTargets(): Promise<DietTargets> {
  return goJson<DietTargets>("/api/diet/targets", { cache: "no-store" });
}

export function setDietTargets(targets: DietTargets): Promise<DietTargets> {
  return goJson<DietTargets>("/api/diet/targets", { method: "PUT", body: JSON.stringify(targets) });
}

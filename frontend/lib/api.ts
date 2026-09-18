import "server-only";
import { goJson } from "./serverFetch";
import type {
  Category,
  CreditCard,
  MonthSummary,
  CreateTransactionInput,
} from "./types";

// The address of the Go API, and the viewer's bearer token, are both read
// inside lib/serverFetch.ts; this module is still marked server-only so a
// client component that accidentally imports it fails the build instead of
// pulling either of them into the browser bundle. The browser never talks
// to the Go API directly — every request goes through a Next.js server
// component, server action or route handler, which is what lets the Go
// service live entirely behind the mTLS edge with no public listener of its
// own.

export function listCategories(): Promise<Category[]> {
  return goJson<Category[]>("/api/categories", { cache: "no-store" });
}

export function listCreditCards(): Promise<CreditCard[]> {
  return goJson<CreditCard[]>("/api/credit-cards", { cache: "no-store" });
}

export function getMonthSummary(yearMonth: string): Promise<MonthSummary> {
  return goJson<MonthSummary>(`/api/months/${yearMonth}`, { cache: "no-store" });
}

export function createTransaction(input: CreateTransactionInput): Promise<unknown> {
  return goJson("/api/transactions", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateTransaction(id: string, description: string, categoryId: string): Promise<unknown> {
  return goJson(`/api/transactions/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ description, category_id: categoryId }),
  });
}

export function deleteTransaction(id: string): Promise<unknown> {
  return goJson(`/api/transactions/${id}`, { method: "DELETE" });
}

export function updateCategoryBudget(id: string, monthlyBudgetCents: number | null): Promise<Category> {
  return goJson<Category>(`/api/categories/${id}/budget`, {
    method: "PATCH",
    body: JSON.stringify({ monthly_budget_cents: monthlyBudgetCents }),
  });
}

export interface CategoryInput {
  name: string;
  nature: Category["nature"];
  color: string;
}

export function createCategory(input: CategoryInput): Promise<Category> {
  return goJson<Category>("/api/categories", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateCategory(id: string, input: CategoryInput): Promise<Category> {
  return goJson<Category>(`/api/categories/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteCategory(id: string): Promise<unknown> {
  return goJson(`/api/categories/${id}`, { method: "DELETE" });
}

export interface CreditCardInput {
  name: string;
  closing_day: number;
  due_day: number;
}

export function createCreditCard(input: CreditCardInput): Promise<CreditCard> {
  return goJson<CreditCard>("/api/credit-cards", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateCreditCard(id: string, input: CreditCardInput): Promise<CreditCard> {
  return goJson<CreditCard>(`/api/credit-cards/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteCreditCard(id: string): Promise<unknown> {
  return goJson(`/api/credit-cards/${id}`, { method: "DELETE" });
}

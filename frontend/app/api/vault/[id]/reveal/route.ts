import { NextResponse } from "next/server";
import { API_URL } from "@/lib/serverFetch";
import { authHeaders } from "@/lib/session";

// Revealing a stored password now costs a second proof that the person at
// the keyboard is still the owner: the Go API wants the account password in
// the body and checks it before it will decrypt anything. That password is
// typed into a modal and posted here, never kept anywhere.
export async function POST(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const body = await request.text();

  const res = await fetch(`${API_URL}/api/vault/${id}/reveal`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...(await authHeaders()) },
    body,
    cache: "no-store",
  });
  const responseBody = await res.text();
  return new NextResponse(responseBody, {
    status: res.status,
    headers: { "Content-Type": "application/json; charset=utf-8" },
  });
}

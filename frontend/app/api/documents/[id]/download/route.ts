import { NextResponse } from "next/server";
import { API_URL } from "@/lib/serverFetch";
import { authHeaders } from "@/lib/session";

// The browser never talks to the Go API directly, so a download is proxied
// here: the file streams through this handler rather than being buffered,
// and the API stays unreachable from the outside.

export async function GET(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const res = await fetch(`${API_URL}/api/documents/${id}/content`, {
    headers: await authHeaders(),
    cache: "no-store",
  });

  if (!res.ok || !res.body) {
    return NextResponse.json({ error: "Arquivo não encontrado." }, { status: res.status === 404 ? 404 : 502 });
  }

  const headers = new Headers();
  for (const header of ["content-type", "content-length", "content-disposition"]) {
    const value = res.headers.get(header);
    if (value) headers.set(header, value);
  }
  return new NextResponse(res.body, { status: 200, headers });
}

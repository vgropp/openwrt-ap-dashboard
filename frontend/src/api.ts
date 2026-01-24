import type { Client } from "./types";

const API_BASE = "./api";

export async function fetchClients(): Promise<Client[]> {
  const res = await fetch(`${API_BASE}/clients`);
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    console.error(`[api] fetchClients failed: ${res.status} ${res.statusText} ${body}`);
    throw new Error(`Failed to fetch clients: ${res.status} ${res.statusText} ${body}`);
  }

  const ct = (res.headers.get("content-type") || "").toLowerCase();
  if (!ct.includes("application/json")) {
    const body = await res.text().catch(() => "");
    console.error(`[api] fetchClients unexpected content-type: ${ct} (body=${body})`);
    throw new Error(`Unexpected content-type: ${ct} (body=${body})`);
  }

  let data: unknown;
  try {
    data = await res.json();
  } catch (err) {
    console.error(`[api] fetchClients json parse error: ${(err as Error).message}`);
    throw new Error(`Failed to parse JSON from /clients: ${(err as Error).message}`);
  }

  if (!Array.isArray(data)) {
    console.error(`[api] fetchClients invalid shape: expected array, got ${typeof data}`);
    throw new Error(`Invalid response shape for clients: expected array`);
  }

  return data as Client[];
}

export async function disconnectClient(mac: string): Promise<void> {
  const url = `${API_BASE}/clients/${encodeURIComponent(mac)}/disconnect`;
  const res = await fetch(url, { method: "POST" });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    console.error(`[api] disconnectClient failed: ${res.status} ${res.statusText} ${body}`);
    throw new Error(`Failed to disconnect client: ${res.status} ${res.statusText} ${body}`);
  }

  // if endpoint returns JSON on success, attempt parse and log only on parse error
  const ct = (res.headers.get("content-type") || "").toLowerCase();
  if (ct && ct.includes("application/json")) {
    try {
      await res.json();
    } catch (err) {
      console.error(`[api] disconnectClient json parse error on success response: ${(err as Error).message}`);
    }
  }
}

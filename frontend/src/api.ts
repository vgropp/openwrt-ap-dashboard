import type { Client } from "./types";

const API_BASE = "/api";

export async function fetchClients(): Promise<Client[]> {
  const res = await fetch(`${API_BASE}/clients`);
  if (!res.ok) throw new Error("Failed to fetch clients");
  return res.json();
}

export async function disconnectClient(mac: string): Promise<void> {
  const res = await fetch(
    `${API_BASE}/clients/${encodeURIComponent(mac)}/disconnect`,
    { method: "POST" }
  );
  if (!res.ok) throw new Error("Failed to disconnect client");
}

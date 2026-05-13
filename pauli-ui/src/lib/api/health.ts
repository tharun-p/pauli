import { PAULI_API_BASE } from "./config";

export async function healthz(): Promise<boolean> {
  try {
    const res = await fetch(`${PAULI_API_BASE}/healthz`, { cache: "no-store" });
    return res.ok;
  } catch {
    return false;
  }
}

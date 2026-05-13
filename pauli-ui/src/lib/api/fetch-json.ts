import { z } from "zod";
import { PAULI_API_BASE } from "./config";
import { errorBodySchema } from "./schemas";

export class PauliApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code?: string,
  ) {
    super(message);
    this.name = "PauliApiError";
  }
}

function joinUrl(path: string): string {
  const p = path.startsWith("/") ? path : `/${path}`;
  return `${PAULI_API_BASE}${p}`;
}

export async function fetchPauliJson<T>(
  path: string,
  schema: z.ZodType<T>,
  init?: RequestInit,
): Promise<T> {
  const res = await fetch(joinUrl(path), {
    ...init,
    headers: {
      Accept: "application/json",
      ...init?.headers,
    },
    cache: "no-store",
  });

  const text = await res.text();
  let json: unknown;
  try {
    json = text ? JSON.parse(text) : {};
  } catch {
    throw new PauliApiError(`Invalid JSON (${res.status})`, res.status);
  }

  if (!res.ok) {
    const parsed = errorBodySchema.safeParse(json);
    const msg = parsed.success ? parsed.data.error.message : `HTTP ${res.status}`;
    const code = parsed.success ? parsed.data.error.code : undefined;
    throw new PauliApiError(msg, res.status, code);
  }

  const out = schema.safeParse(json);
  if (!out.success) {
    throw new PauliApiError(`Unexpected response shape: ${out.error.message}`, res.status);
  }
  return out.data;
}

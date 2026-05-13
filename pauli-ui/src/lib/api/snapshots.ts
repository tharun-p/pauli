import { z } from "zod";
import { PAULI_API_BASE } from "./config";
import { fetchPauliJson } from "./fetch-json";
import {
  snapshotCountSchema,
  validatorSnapshotListResponseSchema,
  validatorSnapshotSchema,
} from "./schemas";

function joinUrl(path: string): string {
  const p = path.startsWith("/") ? path : `/${path}`;
  return `${PAULI_API_BASE}${p}`;
}

export async function getLatestSnapshot(
  validatorIndex: string | number,
): Promise<z.infer<typeof validatorSnapshotSchema> | null> {
  const res = await fetch(
    joinUrl(`/v1/validators/${validatorIndex}/snapshots/latest`),
    { headers: { Accept: "application/json" }, cache: "no-store" },
  );
  if (res.status === 404) return null;
  const text = await res.text();
  const json = text ? JSON.parse(text) : {};
  if (!res.ok) return null;
  const parsed = validatorSnapshotSchema.safeParse(json);
  return parsed.success ? parsed.data : null;
}

export async function countSnapshots(validatorIndex: string | number): Promise<number> {
  const data = await fetchPauliJson(
    `/v1/validators/${validatorIndex}/snapshots/count`,
    snapshotCountSchema,
  );
  return data.count;
}

export type SnapshotSlotQuery = {
  fromSlot: number;
  toSlot: number;
  limit?: number;
  offset?: number;
};

export function buildSnapshotListQueryString(q: SnapshotSlotQuery): string {
  const p = new URLSearchParams({
    from_slot: String(q.fromSlot),
    to_slot: String(q.toSlot),
    limit: String(q.limit ?? 1000),
    offset: String(q.offset ?? 0),
  });
  return p.toString();
}

export async function listSnapshotsPage(
  validatorIndex: string | number,
  q: SnapshotSlotQuery,
) {
  const qs = buildSnapshotListQueryString(q);
  return fetchPauliJson(
    `/v1/validators/${validatorIndex}/snapshots?${qs}`,
    validatorSnapshotListResponseSchema,
  );
}

/** Paginates newest-first pages until empty or maxRows. */
export async function fetchAllSnapshotsInSlotRange(
  validatorIndex: string | number,
  fromSlot: number,
  toSlot: number,
  maxRows = 15_000,
): Promise<z.infer<typeof validatorSnapshotSchema>[]> {
  const limit = 1000;
  const rows: z.infer<typeof validatorSnapshotSchema>[] = [];
  for (let offset = 0; rows.length < maxRows; offset += limit) {
    const page = await listSnapshotsPage(validatorIndex, {
      fromSlot,
      toSlot,
      limit,
      offset,
    });
    rows.push(...page.data);
    if (page.data.length < limit) break;
  }
  return rows;
}

import { fetchPauliJson } from "./fetch-json";
import {
  validatorListResponseSchema,
  type ValidatorIndexRow,
} from "./schemas";

const PAGE_LIMIT = 1000;
const MAX_PAGES = 50;

export async function listValidators(
  limit: number,
  offset: number,
): Promise<{ data: ValidatorIndexRow[]; meta: { limit: number; offset: number; count: number } }> {
  const q = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  return fetchPauliJson(`/v1/validators?${q}`, validatorListResponseSchema);
}

/** Walks paginated /v1/validators until empty or caps — for dashboard totals. */
export async function countAllValidatorsWithSnapshots(): Promise<number> {
  let total = 0;
  for (let page = 0; page < MAX_PAGES; page++) {
    const { data, meta } = await listValidators(PAGE_LIMIT, page * PAGE_LIMIT);
    total += data.length;
    if (data.length < meta.limit || data.length === 0) break;
  }
  return total;
}

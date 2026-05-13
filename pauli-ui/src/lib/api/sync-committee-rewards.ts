import { fetchPauliJson } from "./fetch-json";
import {
  syncCommitteeRewardListResponseSchema,
  type SyncCommitteeRewardRow,
} from "./schemas";
import type { SlotWindowQuery } from "./slot-query";

function buildSlotParams(q: SlotWindowQuery): string {
  const p = new URLSearchParams({
    from_slot: String(q.fromSlot),
    to_slot: String(q.toSlot),
    limit: String(q.limit ?? 500),
    offset: String(q.offset ?? 0),
  });
  if (q.validatorIndex != null) p.set("validator_index", String(q.validatorIndex));
  return p.toString();
}

export async function listSyncCommitteeRewardsGlobal(q: SlotWindowQuery) {
  return fetchPauliJson(
    `/v1/sync-committee-rewards?${buildSlotParams(q)}`,
    syncCommitteeRewardListResponseSchema,
  );
}

export async function listSyncCommitteeRewardsScoped(
  validatorIndex: string | number,
  q: SlotWindowQuery,
) {
  return fetchPauliJson(
    `/v1/validators/${validatorIndex}/sync-committee-rewards?${buildSlotParams(q)}`,
    syncCommitteeRewardListResponseSchema,
  );
}

export async function fetchAllSyncCommitteeRewards(
  q: Omit<SlotWindowQuery, "offset" | "limit">,
  scopedValidator?: string | number,
  maxRows = 10_000,
): Promise<SyncCommitteeRewardRow[]> {
  const limit = 1000;
  const rows: SyncCommitteeRewardRow[] = [];
  for (let offset = 0; rows.length < maxRows; offset += limit) {
    const batch =
      scopedValidator != null
        ? await listSyncCommitteeRewardsScoped(scopedValidator, { ...q, limit, offset })
        : await listSyncCommitteeRewardsGlobal({ ...q, limit, offset });
    rows.push(...batch.data);
    if (batch.data.length < limit) break;
  }
  return rows;
}

import { fetchPauliJson } from "./fetch-json";
import {
  blockProposerRewardListResponseSchema,
  type BlockProposerRewardRow,
} from "./schemas";
import type { SlotWindowQuery } from "./slot-query";

export type { SlotWindowQuery } from "./slot-query";

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

export async function listBlockProposerRewardsGlobal(q: SlotWindowQuery) {
  return fetchPauliJson(
    `/v1/block-proposer-rewards?${buildSlotParams(q)}`,
    blockProposerRewardListResponseSchema,
  );
}

export async function listBlockProposerRewardsScoped(
  validatorIndex: string | number,
  q: SlotWindowQuery,
) {
  return fetchPauliJson(
    `/v1/validators/${validatorIndex}/block-proposer-rewards?${buildSlotParams(q)}`,
    blockProposerRewardListResponseSchema,
  );
}

export async function fetchAllBlockProposerRewards(
  q: Omit<SlotWindowQuery, "offset" | "limit">,
  scopedValidator?: string | number,
  maxRows = 10_000,
): Promise<BlockProposerRewardRow[]> {
  const limit = 1000;
  const rows: BlockProposerRewardRow[] = [];
  for (let offset = 0; rows.length < maxRows; offset += limit) {
    const batch =
      scopedValidator != null
        ? await listBlockProposerRewardsScoped(scopedValidator, { ...q, limit, offset })
        : await listBlockProposerRewardsGlobal({ ...q, limit, offset });
    rows.push(...batch.data);
    if (batch.data.length < limit) break;
  }
  return rows;
}

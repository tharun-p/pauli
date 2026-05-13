import type { AttestationRewardRow } from "@/lib/api/schemas";

/** Sum total_reward across all validators per epoch (for global charts). */
export function aggregateTotalRewardByEpoch(
  rows: AttestationRewardRow[],
): { epoch: number; totalGwei: number }[] {
  const map = new Map<number, number>();
  for (const r of rows) {
    map.set(r.epoch, (map.get(r.epoch) ?? 0) + r.total_reward);
  }
  return [...map.entries()]
    .sort((a, b) => a[0] - b[0])
    .map(([epoch, totalGwei]) => ({ epoch, totalGwei }));
}

/** Per-epoch line for one validator (already scoped rows). */
export function rewardsByEpochForValidator(rows: AttestationRewardRow[]) {
  const map = new Map<number, AttestationRewardRow>();
  for (const r of rows) {
    const prev = map.get(r.epoch);
    if (!prev || r.timestamp > prev.timestamp) map.set(r.epoch, r);
  }
  return [...map.entries()]
    .sort((a, b) => a[0] - b[0])
    .map(([epoch, row]) => ({
      epoch,
      totalGwei: row.total_reward,
      head: row.head_reward,
      source: row.source_reward,
      target: row.target_reward,
    }));
}

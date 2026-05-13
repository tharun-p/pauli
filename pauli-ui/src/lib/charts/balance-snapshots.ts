import type { ValidatorSnapshot } from "@/lib/api/schemas";

export const SLOTS_PER_EPOCH = 32;

export function sortSnapshotsAscBySlot(rows: ValidatorSnapshot[]): ValidatorSnapshot[] {
  return [...rows].sort((a, b) => a.slot - b.slot);
}

export type BalanceEpochPoint = {
  epoch: number;
  balance: number;
  slot: number;
};

/** One point per epoch: balance from the newest snapshot in that epoch (by slot). */
export function aggregateBalanceByEpoch(rowsAsc: ValidatorSnapshot[]): BalanceEpochPoint[] {
  const byEpoch = new Map<number, ValidatorSnapshot>();
  for (const row of rowsAsc) {
    const epoch = Math.floor(row.slot / SLOTS_PER_EPOCH);
    const prev = byEpoch.get(epoch);
    if (!prev || row.slot > prev.slot) {
      byEpoch.set(epoch, row);
    }
  }
  return Array.from(byEpoch.entries())
    .sort(([a], [b]) => a - b)
    .map(([epoch, snap]) => ({ epoch, balance: snap.balance, slot: snap.slot }));
}

export type BalanceSlotPoint = {
  slot: number;
  balance: number;
};

export function snapshotsToSlotSeries(rowsAsc: ValidatorSnapshot[]): BalanceSlotPoint[] {
  return rowsAsc.map((r) => ({ slot: r.slot, balance: r.balance }));
}

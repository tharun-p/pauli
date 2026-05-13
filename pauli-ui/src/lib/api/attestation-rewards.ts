import { fetchPauliJson } from "./fetch-json";
import { attestationRewardListResponseSchema } from "./schemas";

export type AttestationQuery = {
  fromEpoch: number;
  toEpoch: number;
  validatorIndex?: number;
  limit?: number;
  offset?: number;
};

export function buildAttestationQueryString(q: AttestationQuery): string {
  const p = new URLSearchParams({
    from_epoch: String(q.fromEpoch),
    to_epoch: String(q.toEpoch),
    limit: String(q.limit ?? 1000),
    offset: String(q.offset ?? 0),
  });
  if (q.validatorIndex != null) {
    p.set("validator_index", String(q.validatorIndex));
  }
  return p.toString();
}

export async function listAttestationRewardsGlobal(q: AttestationQuery) {
  const qs = buildAttestationQueryString(q);
  return fetchPauliJson(`/v1/attestation-rewards?${qs}`, attestationRewardListResponseSchema);
}

export async function listAttestationRewardsScoped(
  validatorIndex: string | number,
  q: AttestationQuery,
) {
  const qs = buildAttestationQueryString(q);
  return fetchPauliJson(
    `/v1/validators/${validatorIndex}/attestation-rewards?${qs}`,
    attestationRewardListResponseSchema,
  );
}

/** Fetches all pages up to maxRows for charting / tables. */
export async function fetchAllAttestationRewards(
  q: Omit<AttestationQuery, "offset" | "limit"> & { maxRows?: number },
  scopedValidator?: string | number,
): Promise<import("./schemas").AttestationRewardRow[]> {
  const limit = 1000;
  const maxRows = q.maxRows ?? 25_000;
  const rows: import("./schemas").AttestationRewardRow[] = [];
  for (let offset = 0; rows.length < maxRows; offset += limit) {
    const batch = scopedValidator != null
      ? await listAttestationRewardsScoped(scopedValidator, { ...q, limit, offset })
      : await listAttestationRewardsGlobal({ ...q, limit, offset });
    rows.push(...batch.data);
    if (batch.data.length < limit) break;
  }
  return rows;
}

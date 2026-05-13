import { fetchPauliJson } from "./fetch-json";
import { penaltyListResponseSchema, type PenaltyRow } from "./schemas";
import type { AttestationQuery } from "./attestation-rewards";
import { buildAttestationQueryString } from "./attestation-rewards";

export async function listPenalties(
  validatorIndex: string | number,
  q: AttestationQuery,
) {
  const qs = buildAttestationQueryString(q);
  return fetchPauliJson(
    `/v1/validators/${validatorIndex}/penalties?${qs}`,
    penaltyListResponseSchema,
  );
}

export async function fetchAllPenalties(
  validatorIndex: string | number,
  q: Omit<AttestationQuery, "offset" | "limit">,
  maxRows = 5000,
): Promise<PenaltyRow[]> {
  const limit = 1000;
  const rows: PenaltyRow[] = [];
  for (let offset = 0; rows.length < maxRows; offset += limit) {
    const batch = await listPenalties(validatorIndex, { ...q, limit, offset });
    rows.push(...batch.data);
    if (batch.data.length < limit) break;
  }
  return rows;
}

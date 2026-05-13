import { z } from "zod";

export const listMetaSchema = z.object({
  limit: z.number(),
  offset: z.number(),
  count: z.number(),
});

export const errorBodySchema = z.object({
  error: z.object({
    code: z.string(),
    message: z.string(),
  }),
});

export const validatorIndexRowSchema = z.object({
  validator_index: z.coerce.number(),
});

export const validatorListResponseSchema = z.object({
  data: z.array(validatorIndexRowSchema),
  meta: listMetaSchema,
});

export const validatorSnapshotSchema = z.object({
  validator_index: z.coerce.number(),
  slot: z.coerce.number(),
  status: z.string(),
  balance: z.coerce.number(),
  effective_balance: z.coerce.number(),
  timestamp: z.string(),
});

export const snapshotCountSchema = z.object({
  count: z.number(),
});

export const validatorSnapshotListResponseSchema = z.object({
  data: z.array(validatorSnapshotSchema),
  meta: listMetaSchema,
});

export const attestationRewardRowSchema = z.object({
  validator_index: z.coerce.number(),
  epoch: z.coerce.number(),
  head_reward: z.coerce.number(),
  source_reward: z.coerce.number(),
  target_reward: z.coerce.number(),
  total_reward: z.coerce.number(),
  timestamp: z.string(),
});

export const attestationRewardListResponseSchema = z.object({
  data: z.array(attestationRewardRowSchema),
  meta: listMetaSchema,
});

export const blockProposerRewardRowSchema = z.object({
  validator_index: z.coerce.number(),
  validator_pubkey: z.string(),
  slot_number: z.coerce.number(),
  block_number: z.union([z.coerce.number(), z.null()]).optional(),
  rewards: z.coerce.number(),
  timestamp: z.string(),
});

export const blockProposerRewardListResponseSchema = z.object({
  data: z.array(blockProposerRewardRowSchema),
  meta: listMetaSchema,
});

export const syncCommitteeRewardRowSchema = z.object({
  validator_index: z.coerce.number(),
  slot: z.coerce.number(),
  reward_gwei: z.coerce.number(),
  execution_optimistic: z.boolean(),
  finalized: z.boolean(),
  timestamp: z.string(),
});

export const syncCommitteeRewardListResponseSchema = z.object({
  data: z.array(syncCommitteeRewardRowSchema),
  meta: listMetaSchema,
});

export const penaltyRowSchema = z.object({
  validator_index: z.coerce.number(),
  epoch: z.coerce.number(),
  slot: z.coerce.number(),
  penalty_type: z.string(),
  penalty_gwei: z.coerce.number(),
  timestamp: z.string(),
});

export const penaltyListResponseSchema = z.object({
  data: z.array(penaltyRowSchema),
  meta: listMetaSchema,
});

export type ValidatorIndexRow = z.infer<typeof validatorIndexRowSchema>;
export type ValidatorSnapshot = z.infer<typeof validatorSnapshotSchema>;
export type ValidatorSnapshotListResponse = z.infer<typeof validatorSnapshotListResponseSchema>;
export type AttestationRewardRow = z.infer<typeof attestationRewardRowSchema>;
export type BlockProposerRewardRow = z.infer<typeof blockProposerRewardRowSchema>;
export type SyncCommitteeRewardRow = z.infer<typeof syncCommitteeRewardRowSchema>;
export type PenaltyRow = z.infer<typeof penaltyRowSchema>;
export type ListMeta = z.infer<typeof listMetaSchema>;

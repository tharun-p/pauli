"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Award, Blocks, RefreshCw } from "lucide-react";
import { ApiErrorAlert } from "@/components/data/ApiErrorAlert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { fetchAllAttestationRewards } from "@/lib/api/attestation-rewards";
import { fetchAllBlockProposerRewards } from "@/lib/api/block-proposer-rewards";
import { fetchAllSyncCommitteeRewards } from "@/lib/api/sync-committee-rewards";
import { formatGweiEth, formatInteger, formatWeiEth } from "@/lib/format";
import { RewardsChartsPanel } from "./rewards-charts";

type Tab = "attestation" | "proposer" | "sync";

type RewardTableColumn = {
  title: string;
  /** Shown under the title; reward units use accent when `subtitleAccent` is true */
  subtitle?: string;
  subtitleAccent?: boolean;
  /** Width hint for `table-fixed` layout (narrow viewports) */
  thClassName?: string;
  /** Extra classes on body cells in this column */
  tdClassName?: string;
  /** When true, cell content is wrapped with `title` for hover (e.g. truncated pubkey) */
  cellTitle?: boolean;
  /** Header and cell horizontal alignment */
  align?: "start" | "end" | "center";
};

type AppliedRewards = {
  fromEpoch: number;
  toEpoch: number;
  fromSlot: number;
  toSlot: number;
  validator?: number;
};

export default function RewardsPage() {
  const [fromEpoch, setFromEpoch] = useState("0");
  const [toEpoch, setToEpoch] = useState("800000");
  const [fromSlot, setFromSlot] = useState("0");
  const [toSlot, setToSlot] = useState("20000000");
  const [validator, setValidator] = useState("");
  const [tab, setTab] = useState<Tab>("attestation");

  const [applied, setApplied] = useState<AppliedRewards>({
    fromEpoch: 0,
    toEpoch: 800_000,
    fromSlot: 0,
    toSlot: 20_000_000,
    validator: undefined,
  });

  const applyFilters = () => {
    const fe = Math.max(0, Number(fromEpoch) || 0);
    const te = Math.max(fe, Number(toEpoch) || 0);
    const fs = Math.max(0, Number(fromSlot) || 0);
    const ts = Math.max(fs, Number(toSlot) || 0);
    const v = validator.trim();
    const validatorNum = v === "" ? undefined : Number(v);
    setApplied({
      fromEpoch: fe,
      toEpoch: te,
      fromSlot: fs,
      toSlot: ts,
      validator: Number.isFinite(validatorNum) ? validatorNum : undefined,
    });
  };

  const attestationQ = useQuery({
    queryKey: [
      "rewards-attestation",
      applied.fromEpoch,
      applied.toEpoch,
      applied.validator ?? "all",
    ],
    queryFn: () =>
      fetchAllAttestationRewards(
        { fromEpoch: applied.fromEpoch, toEpoch: applied.toEpoch, maxRows: 15_000 },
        applied.validator,
      ),
  });

  const proposerQ = useQuery({
    queryKey: [
      "rewards-proposer",
      applied.fromSlot,
      applied.toSlot,
      applied.validator ?? "all",
    ],
    queryFn: () =>
      fetchAllBlockProposerRewards(
        {
          fromSlot: applied.fromSlot,
          toSlot: applied.toSlot,
          validatorIndex: applied.validator,
        },
        undefined,
        8000,
      ),
  });

  const syncQ = useQuery({
    queryKey: ["rewards-sync", applied.fromSlot, applied.toSlot, applied.validator ?? "all"],
    queryFn: () =>
      fetchAllSyncCommitteeRewards(
        {
          fromSlot: applied.fromSlot,
          toSlot: applied.toSlot,
          validatorIndex: applied.validator,
        },
        undefined,
        8000,
      ),
  });

  const fetching = attestationQ.isFetching || proposerQ.isFetching || syncQ.isFetching;
  const err = attestationQ.error ?? proposerQ.error ?? syncQ.error;
  if (err) return <ApiErrorAlert error={err} />;

  const validatorLabel =
    applied.validator != null ? formatInteger(applied.validator) : "All validators";

  return (
    <div className="mx-auto w-full max-w-[min(100%,1920px)] space-y-8 pb-6">
      <div>
        <h1 className="font-heading text-3xl font-bold tracking-tight md:text-4xl">Rewards & payouts</h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-muted-foreground md:text-base md:leading-relaxed">
          Filter what you load below. <strong className="text-foreground/90">Epoch range</strong> applies to
          attestation rewards. <strong className="text-foreground/90">Slot range</strong> applies to block proposer
          and sync committee rewards. Optional <strong className="text-foreground/90">validator index</strong>{" "}
          scopes every tab to one validator.
        </p>
      </div>

      <div className="glass-panel space-y-8 rounded-xl border border-border/80 p-6 sm:p-7 lg:space-y-10 lg:p-8 xl:p-10">
        <div className="flex flex-col gap-5 border-b border-border/50 pb-6 lg:flex-row lg:items-end lg:justify-between lg:gap-8 lg:pb-8">
          <div className="min-w-0 max-w-3xl space-y-3 lg:max-w-[42rem]">
            <h2 className="font-heading text-xl font-bold tracking-tight text-foreground md:text-2xl lg:text-[1.65rem] lg:leading-tight">
              Query filters
            </h2>
            <p className="text-sm leading-relaxed text-muted-foreground md:text-base md:leading-relaxed">
              These map to API params{" "}
              <code className="rounded-md border border-border/50 bg-muted/40 px-1.5 py-0.5 font-mono text-[0.8rem] text-primary md:text-sm">
                from_epoch
              </code>{" "}
              /{" "}
              <code className="rounded-md border border-border/50 bg-muted/40 px-1.5 py-0.5 font-mono text-[0.8rem] text-primary md:text-sm">
                to_epoch
              </code>
              ,{" "}
              <code className="rounded-md border border-border/50 bg-muted/40 px-1.5 py-0.5 font-mono text-[0.8rem] text-primary md:text-sm">
                from_slot
              </code>{" "}
              /{" "}
              <code className="rounded-md border border-border/50 bg-muted/40 px-1.5 py-0.5 font-mono text-[0.8rem] text-primary md:text-sm">
                to_slot
              </code>
              , and{" "}
              <code className="rounded-md border border-border/50 bg-muted/40 px-1.5 py-0.5 font-mono text-[0.8rem] text-primary md:text-sm">
                validator_index
              </code>
              . Charts and the table use the same loaded rows after you apply.
            </p>
          </div>
          <Button
            type="button"
            onClick={applyFilters}
            disabled={fetching}
            className="h-12 w-full shrink-0 px-6 text-base font-semibold shadow-sm sm:w-auto lg:h-12 lg:min-w-[10.5rem] lg:px-8"
          >
            {fetching ? "Loading…" : "Apply filters"}
          </Button>
        </div>

        <div className="grid gap-6 lg:gap-8 xl:grid-cols-2">
          <section
            aria-labelledby="rewards-epoch-heading"
            className="rounded-2xl border border-border/60 bg-muted/10 p-5 sm:p-6 lg:p-7"
          >
            <h3
              id="rewards-epoch-heading"
              className="mb-5 flex flex-wrap items-center gap-2.5 text-left sm:mb-6"
            >
              <span className="flex size-9 shrink-0 items-center justify-center rounded-xl border border-primary/25 bg-primary/10 text-primary">
                <Award className="size-4" strokeWidth={2} aria-hidden />
              </span>
              <span className="min-w-0">
                <span className="block font-label text-[0.65rem] font-bold uppercase tracking-[0.2em] text-primary sm:text-xs">
                  Epoch range
                </span>
                <span className="block text-sm font-medium text-foreground md:text-base">Attestation rewards</span>
              </span>
            </h3>
            <div className="grid gap-5 sm:grid-cols-2 sm:gap-6">
              <div className="space-y-2.5">
                <Label htmlFor="re-from-epoch" className="text-sm font-semibold text-foreground md:text-[0.9375rem]">
                  From epoch
                </Label>
                <Input
                  id="re-from-epoch"
                  className="h-11 rounded-xl border-border/70 bg-background/60 px-4 py-2 font-mono text-base tracking-tight shadow-inner md:h-12 md:px-4 md:text-lg"
                  value={fromEpoch}
                  onChange={(e) => setFromEpoch(e.target.value)}
                  inputMode="numeric"
                />
                <p className="text-xs leading-snug text-muted-foreground sm:text-sm">Inclusive lower bound.</p>
              </div>
              <div className="space-y-2.5">
                <Label htmlFor="re-to-epoch" className="text-sm font-semibold text-foreground md:text-[0.9375rem]">
                  To epoch
                </Label>
                <Input
                  id="re-to-epoch"
                  className="h-11 rounded-xl border-border/70 bg-background/60 px-4 py-2 font-mono text-base tracking-tight shadow-inner md:h-12 md:px-4 md:text-lg"
                  value={toEpoch}
                  onChange={(e) => setToEpoch(e.target.value)}
                  inputMode="numeric"
                />
              </div>
            </div>
          </section>

          <section
            aria-labelledby="rewards-slot-heading"
            className="rounded-2xl border border-border/60 bg-muted/10 p-5 sm:p-6 lg:p-7"
          >
            <h3
              id="rewards-slot-heading"
              className="mb-5 flex flex-wrap items-center gap-2.5 text-left sm:mb-6"
            >
              <span className="flex size-9 shrink-0 items-center justify-center rounded-xl border border-primary/25 bg-primary/10 text-primary">
                <Blocks className="size-4" strokeWidth={2} aria-hidden />
              </span>
              <span className="min-w-0">
                <span className="block font-label text-[0.65rem] font-bold uppercase tracking-[0.2em] text-primary sm:text-xs">
                  Slot range
                </span>
                <span className="block text-sm font-medium text-foreground md:text-base">
                  Block proposer & sync committee
                </span>
              </span>
            </h3>
            <div className="grid gap-5 sm:grid-cols-2 sm:gap-6">
              <div className="space-y-2.5">
                <Label htmlFor="re-from-slot" className="text-sm font-semibold text-foreground md:text-[0.9375rem]">
                  From slot
                </Label>
                <Input
                  id="re-from-slot"
                  className="h-11 rounded-xl border-border/70 bg-background/60 px-4 py-2 font-mono text-base tracking-tight shadow-inner md:h-12 md:px-4 md:text-lg"
                  value={fromSlot}
                  onChange={(e) => setFromSlot(e.target.value)}
                  inputMode="numeric"
                />
                <p className="text-xs leading-snug text-muted-foreground sm:text-sm">Inclusive lower bound.</p>
              </div>
              <div className="space-y-2.5">
                <Label htmlFor="re-to-slot" className="text-sm font-semibold text-foreground md:text-[0.9375rem]">
                  To slot
                </Label>
                <Input
                  id="re-to-slot"
                  className="h-11 rounded-xl border-border/70 bg-background/60 px-4 py-2 font-mono text-base tracking-tight shadow-inner md:h-12 md:px-4 md:text-lg"
                  value={toSlot}
                  onChange={(e) => setToSlot(e.target.value)}
                  inputMode="numeric"
                />
              </div>
            </div>
          </section>
        </div>

        <section
          aria-labelledby="rewards-validator-heading"
          className="rounded-2xl border border-border/60 bg-muted/10 p-5 sm:p-6 lg:p-7"
        >
          <div className="mb-5 flex flex-wrap items-end justify-between gap-4 sm:mb-6 lg:items-center">
            <div className="min-w-0 space-y-1">
              <h3
                id="rewards-validator-heading"
                className="font-label text-[0.65rem] font-bold uppercase tracking-[0.2em] text-primary sm:text-xs"
              >
                Scope
              </h3>
              <Label htmlFor="re-validator" className="text-sm font-semibold text-foreground md:text-[0.9375rem]">
                Validator index <span className="font-normal text-muted-foreground">(optional)</span>
              </Label>
            </div>
          </div>
          <Input
            id="re-validator"
            className="max-w-full font-mono text-base tracking-tight sm:max-w-xl md:max-w-2xl lg:max-w-3xl xl:max-w-[36rem] h-11 rounded-xl border-border/70 bg-background/60 px-4 py-2 shadow-inner md:h-12 md:px-4 md:text-lg"
            placeholder="Leave empty for all validators"
            value={validator}
            onChange={(e) => setValidator(e.target.value)}
            inputMode="numeric"
          />
        </section>

        <div
          role="status"
          className="flex flex-col gap-1 rounded-xl border border-border/60 bg-muted/25 px-4 py-3.5 text-sm text-muted-foreground sm:flex-row sm:flex-wrap sm:items-baseline sm:gap-x-2 sm:gap-y-1 sm:px-5 sm:py-4 md:text-base"
        >
          <span className="font-semibold text-foreground">Active query</span>
          <span className="hidden sm:inline text-muted-foreground/40" aria-hidden>
            ·
          </span>
          <span>
            Epochs{" "}
            <span className="font-mono font-semibold tabular-nums text-primary">{formatInteger(applied.fromEpoch)}</span>
            <span className="text-muted-foreground">–</span>
            <span className="font-mono font-semibold tabular-nums text-primary">{formatInteger(applied.toEpoch)}</span>
          </span>
          <span className="hidden sm:inline text-muted-foreground/40" aria-hidden>
            ·
          </span>
          <span>
            Slots{" "}
            <span className="font-mono font-semibold tabular-nums text-primary">{formatInteger(applied.fromSlot)}</span>
            <span className="text-muted-foreground">–</span>
            <span className="font-mono font-semibold tabular-nums text-primary">{formatInteger(applied.toSlot)}</span>
          </span>
          <span className="hidden sm:inline text-muted-foreground/40" aria-hidden>
            ·
          </span>
          <span className="font-mono font-semibold text-primary/95">{validatorLabel}</span>
        </div>
      </div>

      {/* Two-column layout must wrap Tabs, not live inside Tabs.Root — Base UI applies data-horizontal:flex-col on the root. */}
      <div className="flex min-h-0 w-full flex-col gap-8 lg:flex-row lg:items-start lg:gap-10 xl:gap-12">
        <div className="min-h-0 min-w-0 flex-1 space-y-5 lg:min-w-0">
          <Tabs value={tab} onValueChange={(v) => setTab(v as Tab)} className="flex w-full flex-col gap-5">
            <TabsList
              variant="default"
              className="flex h-auto min-h-0 w-full max-w-none flex-wrap items-stretch justify-center gap-2 rounded-xl border border-border/50 bg-muted/50 p-2 shadow-inner !h-auto md:gap-2.5 md:p-2.5"
            >
              <TabsTrigger
                value="attestation"
                className={cn(
                  "relative min-h-11 flex-1 rounded-xl border border-transparent px-4 py-2.5 text-sm font-semibold shadow-none transition-all md:min-h-12 md:px-5 md:text-base",
                  "text-muted-foreground hover:border-border/60 hover:bg-background/40 hover:text-foreground",
                  "data-active:border-primary/45 data-active:bg-primary data-active:text-primary-foreground",
                  "data-active:shadow-[0_4px_28px_-6px_rgba(223,255,0,0.4)]",
                  "focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                  "data-active:[&_svg]:text-primary-foreground",
                )}
              >
                <Award className="mr-2 size-4 shrink-0 opacity-90 md:size-[1.125rem]" strokeWidth={2} />
                Attestation
              </TabsTrigger>
              <TabsTrigger
                value="proposer"
                className={cn(
                  "relative min-h-11 flex-1 rounded-xl border border-transparent px-4 py-2.5 text-sm font-semibold shadow-none transition-all md:min-h-12 md:px-5 md:text-base",
                  "text-muted-foreground hover:border-border/60 hover:bg-background/40 hover:text-foreground",
                  "data-active:border-primary/45 data-active:bg-primary data-active:text-primary-foreground",
                  "data-active:shadow-[0_4px_28px_-6px_rgba(223,255,0,0.4)]",
                  "focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                  "data-active:[&_svg]:text-primary-foreground",
                )}
              >
                <Blocks className="mr-2 size-4 shrink-0 opacity-90 md:size-[1.125rem]" strokeWidth={2} />
                Block proposer
              </TabsTrigger>
              <TabsTrigger
                value="sync"
                className={cn(
                  "relative min-h-11 flex-1 rounded-xl border border-transparent px-4 py-2.5 text-sm font-semibold shadow-none transition-all md:min-h-12 md:px-5 md:text-base",
                  "text-muted-foreground hover:border-border/60 hover:bg-background/40 hover:text-foreground",
                  "data-active:border-primary/45 data-active:bg-primary data-active:text-primary-foreground",
                  "data-active:shadow-[0_4px_28px_-6px_rgba(223,255,0,0.4)]",
                  "focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                  "data-active:[&_svg]:text-primary-foreground",
                )}
              >
                <RefreshCw className="mr-2 size-4 shrink-0 opacity-90 md:size-[1.125rem]" strokeWidth={2} />
                Sync committee
              </TabsTrigger>
            </TabsList>

            <TabsContent value="attestation" className="mt-0 min-h-0">
              {attestationQ.isPending ? (
                <Skeleton className="h-[min(28rem,55vh)] w-full rounded-xl" />
              ) : (
                <RewardTable
                  rows={(attestationQ.data ?? []).slice(0, 500).map((r) => ({
                    key: `${r.validator_index}-${r.epoch}`,
                    cells: [
                      formatInteger(r.validator_index),
                      formatInteger(r.epoch),
                      formatInteger(r.head_reward),
                      formatInteger(r.source_reward),
                      formatInteger(r.target_reward),
                      formatInteger(r.total_reward),
                    ],
                  }))}
                  columns={[
                    { title: "Validator", thClassName: "w-[14%]" },
                    { title: "Epoch", thClassName: "w-[12%]" },
                    {
                      title: "Head",
                      subtitle: "gwei",
                      subtitleAccent: true,
                      thClassName: "w-[15%]",
                      align: "end",
                    },
                    {
                      title: "Source",
                      subtitle: "gwei",
                      subtitleAccent: true,
                      thClassName: "w-[15%]",
                      align: "end",
                    },
                    {
                      title: "Target",
                      subtitle: "gwei",
                      subtitleAccent: true,
                      thClassName: "w-[15%]",
                      align: "end",
                    },
                    {
                      title: "Total",
                      subtitle: "gwei",
                      subtitleAccent: true,
                      thClassName: "w-[29%]",
                      align: "end",
                    },
                  ]}
                />
              )}
            </TabsContent>

            <TabsContent value="proposer" className="mt-0 min-h-0">
              {proposerQ.isPending ? (
                <Skeleton className="h-[min(28rem,55vh)] w-full rounded-xl" />
              ) : (
                <RewardTable
                  rows={(proposerQ.data ?? []).slice(0, 500).map((r) => ({
                    key: `${r.validator_index}-${r.slot_number}`,
                    cells: [
                      formatInteger(r.validator_index),
                      formatInteger(r.slot_number),
                      formatGweiEth(r.rewards),
                      formatWeiEth(r.execution_priority_fees_wei ?? null),
                    ],
                  }))}
                  columns={[
                    { title: "Validator", thClassName: "w-[18%]" },
                    { title: "Slot", thClassName: "w-[16%]" },
                    {
                      title: "CL reward",
                      subtitle: "ETH",
                      subtitleAccent: true,
                      thClassName: "min-w-0 w-[28%]",
                      align: "end",
                    },
                    {
                      title: "EL tips",
                      subtitle: "ETH",
                      subtitleAccent: true,
                      thClassName: "min-w-0 w-[38%]",
                      align: "end",
                    },
                  ]}
                />
              )}
            </TabsContent>

            <TabsContent value="sync" className="mt-0 min-h-0">
              {syncQ.isPending ? (
                <Skeleton className="h-[min(28rem,55vh)] w-full rounded-xl" />
              ) : (
                <RewardTable
                  rows={(syncQ.data ?? []).slice(0, 500).map((r) => ({
                    key: `${r.validator_index}-${r.slot}`,
                    cells: [
                      formatInteger(r.validator_index),
                      formatInteger(r.slot),
                      formatInteger(r.reward_gwei),
                    ],
                  }))}
                  columns={[
                    { title: "Validator", thClassName: "w-[28%]" },
                    { title: "Slot", thClassName: "w-[26%]" },
                    {
                      title: "Reward",
                      subtitle: "gwei",
                      subtitleAccent: true,
                      thClassName: "min-w-0 w-[46%]",
                      align: "end",
                    },
                  ]}
                />
              )}
            </TabsContent>
          </Tabs>
        </div>

        <aside className="w-full shrink-0 space-y-4 border-t border-border/40 pt-8 lg:w-[min(100%,440px)] lg:border-l lg:border-t-0 lg:pl-10 lg:pt-0 xl:w-[min(100%,480px)] xl:pl-12">
          <div className="lg:sticky lg:top-6 xl:top-8">
            <h3 className="font-label text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
              Charts · same filters as table
            </h3>
            <p className="mt-1 text-xs leading-relaxed text-muted-foreground/90">
              Updates when you switch tabs. Uses the rows returned for the active query.
            </p>
            <div className="mt-5">
              <RewardsChartsPanel
                tab={tab}
                loading={fetching || attestationQ.isPending || proposerQ.isPending || syncQ.isPending}
                attestationRows={attestationQ.data ?? []}
                proposerRows={proposerQ.data ?? []}
                syncRows={syncQ.data ?? []}
              />
            </div>
          </div>
        </aside>
      </div>
    </div>
  );
}

function RewardTable({
  columns,
  rows,
}: {
  columns: RewardTableColumn[];
  rows: { key: string; cells: string[] }[];
}) {
  return (
    <div
      className={cn(
        "glass-panel w-full min-w-0 overflow-auto rounded-xl border border-border/80 shadow-sm",
        "max-h-[min(70vh,520px)] md:max-h-[min(72vh,600px)] lg:max-h-[min(80vh,720px)] xl:max-h-[calc(100vh-11rem)]",
      )}
    >
      <Table className="w-full min-w-0 table-fixed text-[0.8125rem] leading-snug md:table-auto md:text-sm lg:text-[0.9375rem] lg:leading-normal">
        <TableHeader>
          <TableRow className="border-border/60 hover:bg-transparent">
            {columns.map((col) => (
              <TableHead
                key={`${col.title}-${col.subtitle ?? ""}`}
                className={cn(
                  "min-w-0 align-top whitespace-normal break-words px-3 py-3 font-label text-[0.62rem] font-semibold uppercase leading-tight tracking-[0.1em] text-muted-foreground md:px-4 md:py-3.5 lg:text-[0.7rem]",
                  col.align === "end" && "text-right",
                  col.align === "center" && "text-center",
                  col.thClassName,
                )}
              >
                <span className={cn("block break-words", col.align === "end" && "text-right")}>{col.title}</span>
                {col.subtitle ? (
                  <span
                    className={cn(
                      "mt-1 block text-[0.58rem] font-medium leading-snug tracking-normal normal-case sm:text-[0.6rem]",
                      col.subtitleAccent ? "text-primary" : "text-muted-foreground/90",
                      col.align === "end" && "text-right",
                      col.align === "center" && "text-center",
                    )}
                  >
                    {col.subtitle}
                  </span>
                ) : null}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.key} className="border-border/40 text-foreground/95">
              {r.cells.map((c, i) => {
                const col = columns[i];
                return (
                  <TableCell
                    key={i}
                    className={cn(
                      "min-w-0 align-middle px-3 py-2.5 md:px-4 md:py-3",
                      i < 2 && "font-mono text-[0.875rem] font-semibold text-primary md:text-base",
                      i >= 2 &&
                        "break-words font-sans tabular-nums tracking-tight text-foreground md:whitespace-nowrap",
                      col?.align === "end" && "text-right",
                      col?.align === "center" && "text-center",
                      col?.tdClassName,
                    )}
                  >
                    {col?.cellTitle ? <span title={c}>{c}</span> : c}
                  </TableCell>
                );
              })}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

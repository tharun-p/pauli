"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Layers, ListOrdered } from "lucide-react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { ChartCard } from "@/components/charts/ChartCard";
import { chartTheme } from "@/components/charts/chart-theme";
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
import { fetchAllSnapshotsInSlotRange } from "@/lib/api/snapshots";
import { listValidators } from "@/lib/api/validators";
import {
  aggregateBalanceByEpoch,
  snapshotsToSlotSeries,
  sortSnapshotsAscBySlot,
} from "@/lib/charts/balance-snapshots";
import { formatGweiEth, formatInteger } from "@/lib/format";

const MAX_SNAPSHOT_ROWS = 15_000;

type View = "slot" | "epoch";

type Applied = {
  validatorIndex: number;
  fromSlot: number;
  toSlot: number;
};

async function loadBalances(applied: Applied) {
  const raw = await fetchAllSnapshotsInSlotRange(
    applied.validatorIndex,
    applied.fromSlot,
    applied.toSlot,
    MAX_SNAPSHOT_ROWS,
  );
  const asc = sortSnapshotsAscBySlot(raw);
  const bySlot = snapshotsToSlotSeries(asc);
  const byEpoch = aggregateBalanceByEpoch(asc);
  return {
    truncated: raw.length >= MAX_SNAPSHOT_ROWS,
    rowCount: raw.length,
    bySlot,
    byEpoch,
    ascSnapshots: asc,
  };
}

export default function BalancesPage() {
  const [validator, setValidator] = useState("");
  const [fromSlot, setFromSlot] = useState("0");
  const [toSlot, setToSlot] = useState("20000000");
  const [view, setView] = useState<View>("epoch");
  const [applied, setApplied] = useState<Applied | null>(null);

  const seedQ = useQuery({
    queryKey: ["balances-seed-validator"],
    queryFn: async () => {
      const page = await listValidators(1, 0);
      const first = page.data[0]?.validator_index;
      return first != null ? String(first) : "";
    },
  });

  const apply = () => {
    const trimmed = validator.trim();
    let v: number;
    if (trimmed === "") {
      const seed = seedQ.data?.trim() ?? "";
      v = seed !== "" ? Number(seed) : 0;
    } else {
      v = Number(trimmed);
      if (!Number.isFinite(v) || v < 0) v = 0;
    }
    const fs = Math.max(0, Number(fromSlot) || 0);
    const ts = Math.max(fs, Number(toSlot) || 0);
    setApplied({ validatorIndex: v, fromSlot: fs, toSlot: ts });
  };

  const dataQ = useQuery({
    queryKey: ["balances-series", applied?.validatorIndex, applied?.fromSlot, applied?.toSlot],
    queryFn: () => loadBalances(applied!),
    enabled: applied != null,
  });

  if (dataQ.isError) {
    return <ApiErrorAlert error={dataQ.error} />;
  }

  const chartData =
    view === "epoch"
      ? (dataQ.data?.byEpoch ?? []).map((p) => ({
          x: p.epoch,
          balance: p.balance,
          slot: p.slot,
        }))
      : (dataQ.data?.bySlot ?? []).map((p) => ({
          x: p.slot,
          balance: p.balance,
        }));

  const tableRowsEpoch =
    dataQ.data?.byEpoch.slice().reverse().map((p) => ({
      key: `e-${p.epoch}`,
      cells: [
        formatInteger(p.epoch),
        formatInteger(p.slot),
        formatGweiEth(p.balance),
      ],
    })) ?? [];

  const tableRowsSlot =
    dataQ.data?.ascSnapshots
      .slice()
      .reverse()
      .map((p) => ({
        key: `s-${p.slot}`,
        cells: [
          formatInteger(p.slot),
          formatInteger(Math.floor(p.slot / 32)),
          formatGweiEth(p.balance),
          formatGweiEth(p.effective_balance),
          p.status,
          p.timestamp,
        ],
      })) ?? [];

  return (
    <div className="mx-auto w-full max-w-[min(100%,1920px)] space-y-8 pb-6">
      <div>
        <h1 className="font-heading text-3xl font-bold tracking-tight md:text-4xl">Balances</h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-muted-foreground md:text-base md:leading-relaxed">
          Staking balance from indexed snapshots for one validator. Choose a{" "}
          <strong className="text-foreground/90">slot window</strong>, then switch the table and chart
          between <strong className="text-foreground/90">epoch</strong> (latest snapshot per epoch) and{" "}
          <strong className="text-foreground/90">slot</strong> (every stored row).
        </p>
      </div>

      <div className="glass-panel space-y-8 rounded-xl border border-border/80 p-6 sm:p-7 lg:p-8 xl:p-10">
        <div className="flex flex-col gap-5 border-b border-border/50 pb-6 lg:flex-row lg:items-end lg:justify-between lg:gap-8 lg:pb-8">
          <div className="min-w-0 max-w-3xl space-y-3">
            <h2 className="font-heading text-xl font-bold tracking-tight text-foreground md:text-2xl">
              Query filters
            </h2>
            <p className="text-sm leading-relaxed text-muted-foreground md:text-base">
              Maps to{" "}
              <code className="rounded-md border border-border/50 bg-muted/40 px-1.5 py-0.5 font-mono text-[0.8rem] text-primary">
                /v1/validators/{"{index}"}/snapshots
              </code>{" "}
              with <code className="text-primary">from_slot</code> and <code className="text-primary">to_slot</code>.
              Loads up to {formatInteger(MAX_SNAPSHOT_ROWS)} rows.
            </p>
          </div>
          <Button
            type="button"
            onClick={apply}
            disabled={dataQ.isFetching}
            className="h-12 w-full shrink-0 px-6 text-base font-semibold shadow-sm sm:w-auto lg:h-12 lg:min-w-[10.5rem] lg:px-8"
          >
            {dataQ.isFetching ? "Loading…" : "Apply filters"}
          </Button>
        </div>

        <div className="grid gap-6 lg:grid-cols-2">
          <section className="rounded-2xl border border-border/60 bg-muted/10 p-5 sm:p-6 lg:p-7">
            <Label htmlFor="bal-validator" className="text-sm font-semibold text-foreground">
              Validator index
            </Label>
            <Input
              id="bal-validator"
              className="mt-2 h-11 rounded-xl border-border/70 bg-background/60 font-mono text-base md:h-12"
              value={validator}
              onChange={(e) => setValidator(e.target.value)}
              placeholder={seedQ.data ? `Default e.g. ${seedQ.data}` : "0"}
              inputMode="numeric"
            />
            {seedQ.data ? (
              <p className="mt-2 text-xs text-muted-foreground">
                Leave the field empty to use the first indexed validator ({seedQ.data}), or choose another index.
              </p>
            ) : null}
          </section>

          <section className="rounded-2xl border border-border/60 bg-muted/10 p-5 sm:p-6 lg:p-7">
            <div className="grid gap-5 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="bal-from-slot" className="text-sm font-semibold">
                  From slot
                </Label>
                <Input
                  id="bal-from-slot"
                  className="h-11 rounded-xl border-border/70 bg-background/60 font-mono md:h-12"
                  value={fromSlot}
                  onChange={(e) => setFromSlot(e.target.value)}
                  inputMode="numeric"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="bal-to-slot" className="text-sm font-semibold">
                  To slot
                </Label>
                <Input
                  id="bal-to-slot"
                  className="h-11 rounded-xl border-border/70 bg-background/60 font-mono md:h-12"
                  value={toSlot}
                  onChange={(e) => setToSlot(e.target.value)}
                  inputMode="numeric"
                />
              </div>
            </div>
          </section>
        </div>

        {applied ? (
          <div
            role="status"
            className="flex flex-col gap-1 rounded-xl border border-border/60 bg-muted/25 px-4 py-3.5 text-sm text-muted-foreground sm:flex-row sm:flex-wrap sm:items-baseline sm:gap-x-2 sm:px-5 sm:py-4"
          >
            <span className="font-semibold text-foreground">Active query</span>
            <span className="hidden sm:inline text-muted-foreground/40" aria-hidden>
              ·
            </span>
            <span className="font-mono font-semibold text-primary">{formatInteger(applied.validatorIndex)}</span>
            <span className="hidden sm:inline text-muted-foreground/40" aria-hidden>
              ·
            </span>
            <span>
              Slots{" "}
              <span className="font-mono font-semibold tabular-nums text-primary">
                {formatInteger(applied.fromSlot)}
              </span>
              <span className="text-muted-foreground">–</span>
              <span className="font-mono font-semibold tabular-nums text-primary">
                {formatInteger(applied.toSlot)}
              </span>
            </span>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            Set a validator index and slot range, then <strong className="text-foreground">Apply filters</strong> to
            load balances.
          </p>
        )}
      </div>

      {applied && dataQ.isPending ? (
        <div className="space-y-6">
          <Skeleton className="h-[320px] w-full rounded-xl" />
          <Skeleton className="h-80 w-full rounded-xl" />
        </div>
      ) : null}

      {applied && !dataQ.isPending && dataQ.data ? (
        <>
          {dataQ.data.truncated ? (
            <p className="rounded-lg border border-primary/30 bg-primary/5 px-4 py-3 text-sm text-foreground">
              Showing the first {formatInteger(MAX_SNAPSHOT_ROWS)} snapshot rows in this slot window. Narrow the range
              for full coverage.
            </p>
          ) : null}

          <Tabs value={view} onValueChange={(v) => setView(v as View)} className="flex w-full flex-col gap-5">
            <TabsList
              variant="default"
              className="flex h-auto min-h-0 w-full max-w-md flex-wrap gap-2 rounded-xl border border-border/50 bg-muted/50 p-2 !h-auto"
            >
              <TabsTrigger
                value="epoch"
                className={cn(
                  "relative min-h-11 flex-1 rounded-xl border border-transparent px-4 py-2.5 text-sm font-semibold shadow-none transition-all md:min-h-12",
                  "text-muted-foreground hover:border-border/60 hover:bg-background/40 hover:text-foreground",
                  "data-active:border-primary/45 data-active:bg-primary data-active:text-primary-foreground",
                  "data-active:shadow-[0_4px_28px_-6px_rgba(223,255,0,0.4)]",
                  "focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                )}
              >
                <Layers className="mr-2 size-4 shrink-0 opacity-90" strokeWidth={2} />
                By epoch
              </TabsTrigger>
              <TabsTrigger
                value="slot"
                className={cn(
                  "relative min-h-11 flex-1 rounded-xl border border-transparent px-4 py-2.5 text-sm font-semibold shadow-none transition-all md:min-h-12",
                  "text-muted-foreground hover:border-border/60 hover:bg-background/40 hover:text-foreground",
                  "data-active:border-primary/45 data-active:bg-primary data-active:text-primary-foreground",
                  "data-active:shadow-[0_4px_28px_-6px_rgba(223,255,0,0.4)]",
                  "focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                )}
              >
                <ListOrdered className="mr-2 size-4 shrink-0 opacity-90" strokeWidth={2} />
                By slot
              </TabsTrigger>
            </TabsList>

            <ChartCard
              title={view === "epoch" ? "Balance by epoch" : "Balance by slot"}
              description={
                view === "epoch"
                  ? "Balance taken from the newest snapshot in each epoch (highest slot in the epoch)."
                  : "Balance at each stored snapshot row, ordered in time."
              }
            >
              {chartData.length === 0 ? (
                <p className="flex h-full items-center justify-center text-sm text-muted-foreground">
                  No snapshot rows in this window for this validator.
                </p>
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={chartData} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                    <defs>
                      <linearGradient id="fillBalance" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor={chartTheme.primary} stopOpacity={0.35} />
                        <stop offset="100%" stopColor={chartTheme.primary} stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke={chartTheme.grid} opacity={0.5} />
                    <XAxis
                      dataKey="x"
                      tick={{ fill: chartTheme.axis, fontSize: 11 }}
                      tickFormatter={(v) => String(v)}
                    />
                    <YAxis
                      tick={{ fill: chartTheme.axis, fontSize: 11 }}
                      tickFormatter={(v) => formatGweiEth(Number(v), 2)}
                      width={80}
                    />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: chartTheme.tooltipBg,
                        border: `1px solid ${chartTheme.tooltipBorder}`,
                        borderRadius: 8,
                      }}
                      labelFormatter={(x) => (view === "epoch" ? `Epoch ${x}` : `Slot ${x}`)}
                      formatter={(value: unknown) => [
                        formatGweiEth(Number(value ?? 0), 6),
                        "Balance",
                      ]}
                    />
                    <Area
                      type="monotone"
                      dataKey="balance"
                      stroke={chartTheme.primary}
                      fill="url(#fillBalance)"
                      strokeWidth={2}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              )}
            </ChartCard>

            <TabsContent value="epoch" className="mt-0">
              <BalanceTable
                headers={["Epoch", "Slot (latest in epoch)", "Balance"]}
                rows={tableRowsEpoch}
              />
            </TabsContent>
            <TabsContent value="slot" className="mt-0">
              <BalanceTable
                headers={["Slot", "Epoch", "Balance", "Effective", "Status", "Timestamp"]}
                rows={tableRowsSlot}
              />
            </TabsContent>
          </Tabs>
        </>
      ) : null}
    </div>
  );
}

function BalanceTable({
  headers,
  rows,
}: {
  headers: string[];
  rows: { key: string; cells: string[] }[];
}) {
  const lastCol = headers.length - 1;
  return (
    <div
      className={cn(
        "glass-panel w-full overflow-auto rounded-xl border border-border/80 shadow-sm",
        "max-h-[min(70vh,520px)] md:max-h-[min(72vh,600px)]",
      )}
    >
      <Table className="min-w-[640px] text-[0.8125rem] md:min-w-0 md:text-sm">
        <TableHeader>
          <TableRow className="border-border/60 hover:bg-transparent">
            {headers.map((h) => (
              <TableHead
                key={h}
                className="whitespace-nowrap px-3 py-2.5 font-label text-[0.65rem] font-semibold uppercase tracking-[0.14em] text-muted-foreground md:px-4 md:py-3"
              >
                {h}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.key} className="border-border/40 text-foreground/95">
              {r.cells.map((c, i) => (
                <TableCell
                  key={i}
                  className={cn(
                    "whitespace-nowrap px-3 py-2.5 align-middle md:px-4 md:py-3",
                    i < 2 && "font-mono text-[0.875rem] font-semibold text-primary md:text-base",
                    i >= 2 &&
                      i < lastCol &&
                      "font-sans tabular-nums tracking-tight text-foreground",
                    i === lastCol &&
                      "min-w-[11rem] max-w-[24rem] truncate font-sans text-[0.8125rem] text-muted-foreground md:text-sm",
                  )}
                >
                  {c}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

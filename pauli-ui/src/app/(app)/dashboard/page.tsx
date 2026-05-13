"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
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
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { countAllValidatorsWithSnapshots, listValidators } from "@/lib/api/validators";
import { fetchAllAttestationRewards } from "@/lib/api/attestation-rewards";
import { getLatestSnapshot } from "@/lib/api/snapshots";
import type { ValidatorSnapshot } from "@/lib/api/schemas";
import { aggregateTotalRewardByEpoch } from "@/lib/charts/aggregate-attestation";
import { formatGweiEth, formatInteger } from "@/lib/format";
import { cn } from "@/lib/utils";

async function loadDashboardSummary() {
  const totalValidators = await countAllValidatorsWithSnapshots();
  const firstPage = await listValidators(1, 0);
  let fromEpoch = 0;
  let toEpoch = 500_000;
  let sampleValidator: number | null = null;
  let latestSnapshot: ValidatorSnapshot | null = null;
  if (firstPage.data.length > 0) {
    sampleValidator = firstPage.data[0].validator_index;
    const snap = await getLatestSnapshot(sampleValidator);
    latestSnapshot = snap;
    if (snap) {
      toEpoch = Math.floor(Number(snap.slot) / 32);
      fromEpoch = Math.max(0, toEpoch - 100);
    }
  }
  const rows = await fetchAllAttestationRewards(
    { fromEpoch, toEpoch, maxRows: 25_000 },
    undefined,
  );
  const chart = aggregateTotalRewardByEpoch(rows);
  return { totalValidators, fromEpoch, toEpoch, chart, sampleValidator, latestSnapshot };
}

export default function DashboardPage() {
  const q = useQuery({
    queryKey: ["dashboard-summary"],
    queryFn: loadDashboardSummary,
  });

  if (q.isError) {
    return <ApiErrorAlert error={q.error} />;
  }

  if (q.isPending) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <div className="grid gap-4 md:grid-cols-3">
          <Skeleton className="h-28" />
          <Skeleton className="h-28" />
          <Skeleton className="h-28" />
        </div>
        <Skeleton className="h-[320px]" />
      </div>
    );
  }

  const { totalValidators, fromEpoch, toEpoch, chart, sampleValidator, latestSnapshot } = q.data;

  const lastEpoch = chart.length ? chart[chart.length - 1]!.epoch : null;
  const lastEpochRewards = chart.length ? chart[chart.length - 1]!.totalGwei : null;
  const peak =
    chart.length > 0
      ? chart.reduce((a, b) => (a.totalGwei > b.totalGwei ? a : b)).totalGwei
      : 0;
  const peakEpoch =
    chart.length > 0
      ? chart.reduce((a, b) => (a.totalGwei > b.totalGwei ? a : b)).epoch
      : null;

  return (
    <div className="space-y-8">
      <div>
        <h1 className="font-heading text-3xl font-bold tracking-tight text-foreground">
          Staking overview
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Aggregated attestation rewards by epoch (epochs {formatInteger(fromEpoch)}–
          {formatInteger(toEpoch)}).
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card className="glass-panel border-border/80 bg-card/80">
          <CardHeader className="pb-2">
            <CardTitle className="font-label text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Indexed validators
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="font-heading text-3xl font-bold text-primary">
              {formatInteger(totalValidators)}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">Distinct indices with snapshots</p>
          </CardContent>
        </Card>
        <Card className="glass-panel border-border/80 bg-card/80">
          <CardHeader className="pb-2">
            <CardTitle className="font-label text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Chart epochs
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="font-heading text-3xl font-bold text-foreground">
              {chart.length ? formatInteger(chart.length) : "—"}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">Epoch buckets with reward data</p>
          </CardContent>
        </Card>
        <Card className="glass-panel border-border/80 bg-card/80">
          <CardHeader className="pb-2">
            <CardTitle className="font-label text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Peak bucket (Σ gwei)
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="font-heading text-3xl font-bold text-secondary">
              {peak ? formatInteger(peak) : "—"}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {peakEpoch != null ? `Peak at epoch ${formatInteger(peakEpoch)}` : "No rows in window"}
            </p>
          </CardContent>
        </Card>
      </div>

      <ChartCard
        title="Network attestation rewards"
        description="Sum of total_reward (gwei) across all validators per epoch."
      >
        {chart.length === 0 ? (
          <p className="flex h-full items-center justify-center text-sm text-muted-foreground">
            No attestation reward rows for the resolved epoch window.
          </p>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chart} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id="fillPrimary" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={chartTheme.primary} stopOpacity={0.35} />
                  <stop offset="100%" stopColor={chartTheme.primary} stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke={chartTheme.grid} opacity={0.5} />
              <XAxis
                dataKey="epoch"
                tick={{ fill: chartTheme.axis, fontSize: 11 }}
                tickFormatter={(v) => String(v)}
              />
              <YAxis
                tick={{ fill: chartTheme.axis, fontSize: 11 }}
                tickFormatter={(v) => formatGweiEth(Number(v), 2)}
                width={72}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: chartTheme.tooltipBg,
                  border: `1px solid ${chartTheme.tooltipBorder}`,
                  borderRadius: 8,
                }}
                labelFormatter={(e) => `Epoch ${e}`}
                formatter={(value) => [
                  `${formatInteger(Number(value ?? 0))} gwei`,
                  "Σ total_reward",
                ]}
              />
              <Area
                type="monotone"
                dataKey="totalGwei"
                stroke={chartTheme.primary}
                fill="url(#fillPrimary)"
                strokeWidth={2}
              />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </ChartCard>

      <section
        aria-labelledby="dashboard-pulse-heading"
        className="rounded-xl border border-border/60 bg-muted/10 p-5 sm:p-6"
      >
        <div className="flex flex-col gap-4 border-b border-border/40 pb-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2
              id="dashboard-pulse-heading"
              className="font-heading text-lg font-bold tracking-tight text-foreground"
            >
              Rewards & balances pulse
            </h2>
            <p className="mt-1 text-xs text-muted-foreground sm:text-sm">
              Snapshot from the same epoch window as the chart above, plus the latest on-chain balance for the first
              indexed validator.
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Link
              href="/rewards"
              className={cn(buttonVariants({ variant: "outline", size: "sm" }), "rounded-lg border-border/70")}
            >
              Open rewards
            </Link>
            <Link
              href="/balances"
              className={cn(buttonVariants({ variant: "outline", size: "sm" }), "rounded-lg border-border/70")}
            >
              Open balances
            </Link>
          </div>
        </div>

        <div className="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Card className="border-border/60 bg-background/50 shadow-none">
            <CardHeader className="pb-1 pt-4">
              <CardTitle className="font-label text-[0.65rem] font-semibold uppercase tracking-wider text-muted-foreground">
                Last epoch Σ attestation
              </CardTitle>
            </CardHeader>
            <CardContent className="pb-4">
              <p className="font-heading text-xl font-bold tabular-nums text-primary sm:text-2xl">
                {lastEpochRewards != null ? `${formatInteger(lastEpochRewards)} gwei` : "—"}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {lastEpoch != null ? `Epoch ${formatInteger(lastEpoch)}` : "No rows in window"}
              </p>
            </CardContent>
          </Card>

          <Card className="border-border/60 bg-background/50 shadow-none">
            <CardHeader className="pb-1 pt-4">
              <CardTitle className="font-label text-[0.65rem] font-semibold uppercase tracking-wider text-muted-foreground">
                Peak epoch Σ
              </CardTitle>
            </CardHeader>
            <CardContent className="pb-4">
              <p className="font-heading text-xl font-bold tabular-nums text-secondary sm:text-2xl">
                {peak ? `${formatInteger(peak)} gwei` : "—"}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {peakEpoch != null ? `Epoch ${formatInteger(peakEpoch)}` : "—"}
              </p>
            </CardContent>
          </Card>

          <Card className="border-border/60 bg-background/50 shadow-none">
            <CardHeader className="pb-1 pt-4">
              <CardTitle className="font-label text-[0.65rem] font-semibold uppercase tracking-wider text-muted-foreground">
                Latest balance
              </CardTitle>
            </CardHeader>
            <CardContent className="pb-4">
              <p className="font-heading text-xl font-bold tabular-nums text-foreground sm:text-2xl">
                {latestSnapshot ? formatGweiEth(latestSnapshot.balance, 4) : "—"}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {sampleValidator != null ? (
                  <>
                    Validator{" "}
                    <span className="font-mono font-semibold text-primary">{formatInteger(sampleValidator)}</span>
                  </>
                ) : (
                  "No validators indexed"
                )}
              </p>
            </CardContent>
          </Card>

          <Card className="border-border/60 bg-background/50 shadow-none">
            <CardHeader className="pb-1 pt-4">
              <CardTitle className="font-label text-[0.65rem] font-semibold uppercase tracking-wider text-muted-foreground">
                Snapshot slot / epoch
              </CardTitle>
            </CardHeader>
            <CardContent className="pb-4">
              <p className="font-mono text-lg font-semibold text-foreground sm:text-xl">
                {latestSnapshot ? formatInteger(latestSnapshot.slot) : "—"}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {latestSnapshot ? (
                  <>
                    Epoch ~{formatInteger(Math.floor(latestSnapshot.slot / 32))} ·{" "}
                    <span className="truncate">{latestSnapshot.status}</span>
                  </>
                ) : (
                  "No snapshot yet"
                )}
              </p>
            </CardContent>
          </Card>
        </div>
      </section>
    </div>
  );
}

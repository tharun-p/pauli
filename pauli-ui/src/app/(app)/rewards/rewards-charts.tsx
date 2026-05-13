"use client";

import type { ReactNode } from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { chartTheme } from "@/components/charts/chart-theme";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type {
  AttestationRewardRow,
  BlockProposerRewardRow,
  SyncCommitteeRewardRow,
} from "@/lib/api/schemas";
import { formatInteger } from "@/lib/format";

type Tab = "attestation" | "proposer" | "sync";

function aggregateAttestationByEpoch(rows: AttestationRewardRow[]) {
  const m = new Map<number, number>();
  for (const r of rows) {
    m.set(r.epoch, (m.get(r.epoch) ?? 0) + r.total_reward);
  }
  return [...m.entries()]
    .sort((a, b) => a[0] - b[0])
    .map(([epoch, totalGwei]) => ({ epoch, totalGwei }));
}

export function RewardsChartsPanel({
  tab,
  loading,
  attestationRows,
  proposerRows,
  syncRows,
}: {
  tab: Tab;
  loading: boolean;
  attestationRows: AttestationRewardRow[];
  proposerRows: BlockProposerRewardRow[];
  syncRows: SyncCommitteeRewardRow[];
}) {
  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-[220px] w-full rounded-xl lg:h-[260px] xl:h-[280px]" />
        <Skeleton className="h-[200px] w-full rounded-xl lg:h-[240px]" />
      </div>
    );
  }

  if (tab === "attestation") {
    const data = aggregateAttestationByEpoch(attestationRows.slice(0, 5000));
    return (
      <div className="space-y-4">
        <ChartShell
          title="Σ Total reward by epoch"
          description="Sum of total_reward (gwei) for rows in the table — same epoch filter."
        >
          {data.length === 0 ? (
            <EmptyChart />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={chartTheme.grid} opacity={0.45} />
                <XAxis dataKey="epoch" tick={{ fill: chartTheme.axis, fontSize: 10 }} />
                <YAxis
                  tick={{ fill: chartTheme.axis, fontSize: 10 }}
                  tickFormatter={(v) => formatInteger(Number(v))}
                  width={56}
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: chartTheme.tooltipBg,
                    border: `1px solid ${chartTheme.tooltipBorder}`,
                    borderRadius: 8,
                  }}
                  formatter={(v) => [formatInteger(Number(v ?? 0)), "gwei"]}
                  labelFormatter={(e) => `Epoch ${e}`}
                />
                <Bar dataKey="totalGwei" fill={chartTheme.primary} radius={[4, 4, 0, 0]} maxBarSize={48} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </ChartShell>
        <ChartShell
          title="Components (last points)"
          description="Head / source / target totals by epoch (same filtered rows, up to 80 epochs)."
        >
          {attestationRows.length === 0 ? (
            <EmptyChart />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={epochComponentSeries(attestationRows)} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={chartTheme.grid} opacity={0.45} />
                <XAxis dataKey="epoch" tick={{ fill: chartTheme.axis, fontSize: 10 }} />
                <YAxis tick={{ fill: chartTheme.axis, fontSize: 10 }} width={52} tickFormatter={(v) => formatInteger(Number(v))} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: chartTheme.tooltipBg,
                    border: `1px solid ${chartTheme.tooltipBorder}`,
                    borderRadius: 8,
                  }}
                />
                <Line type="monotone" dataKey="head" name="head" stroke={chartTheme.secondary} dot={false} strokeWidth={2} />
                <Line type="monotone" dataKey="source" name="source" stroke={chartTheme.tertiary} dot={false} strokeWidth={2} />
                <Line type="monotone" dataKey="target" name="target" stroke={chartTheme.axis} dot={false} strokeWidth={2} />
              </LineChart>
            </ResponsiveContainer>
          )}
        </ChartShell>
      </div>
    );
  }

  if (tab === "proposer") {
    const data = [...proposerRows]
      .sort((a, b) => Number(a.slot_number) - Number(b.slot_number))
      .slice(0, 400)
      .map((r) => ({ slot: r.slot_number, rewardsGwei: r.rewards }));
    return (
      <div className="space-y-4">
        <ChartShell title="Proposer rewards by slot" description="Rewards (gwei) vs beacon slot — same slot & validator filter.">
          {data.length === 0 ? (
            <EmptyChart />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={chartTheme.grid} opacity={0.45} />
                <XAxis dataKey="slot" tick={{ fill: chartTheme.axis, fontSize: 10 }} tickFormatter={(v) => formatInteger(Number(v))} />
                <YAxis tick={{ fill: chartTheme.axis, fontSize: 10 }} width={52} tickFormatter={(v) => formatInteger(Number(v))} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: chartTheme.tooltipBg,
                    border: `1px solid ${chartTheme.tooltipBorder}`,
                    borderRadius: 8,
                  }}
                  formatter={(v) => [formatInteger(Number(v ?? 0)), "gwei"]}
                  labelFormatter={(s) => `Slot ${s}`}
                />
                <Line type="monotone" dataKey="rewardsGwei" stroke={chartTheme.primary} dot={false} strokeWidth={2} />
              </LineChart>
            </ResponsiveContainer>
          )}
        </ChartShell>
      </div>
    );
  }

  const syncData = [...syncRows]
    .sort((a, b) => Number(a.slot) - Number(b.slot))
    .slice(0, 500)
    .map((r) => ({ slot: r.slot, rewardGwei: r.reward_gwei }));

  return (
    <div className="space-y-4">
      <ChartShell title="Sync committee reward by slot" description="reward_gwei vs slot — same slot & validator filter.">
        {syncData.length === 0 ? (
          <EmptyChart />
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={syncData} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke={chartTheme.grid} opacity={0.45} />
              <XAxis dataKey="slot" tick={{ fill: chartTheme.axis, fontSize: 10 }} tickFormatter={(v) => formatInteger(Number(v))} />
              <YAxis tick={{ fill: chartTheme.axis, fontSize: 10 }} width={52} tickFormatter={(v) => formatInteger(Number(v))} />
              <Tooltip
                contentStyle={{
                  backgroundColor: chartTheme.tooltipBg,
                  border: `1px solid ${chartTheme.tooltipBorder}`,
                  borderRadius: 8,
                }}
                formatter={(v) => [formatInteger(Number(v ?? 0)), "gwei"]}
                labelFormatter={(s) => `Slot ${s}`}
              />
              <Line type="monotone" dataKey="rewardGwei" stroke={chartTheme.primary} dot={false} strokeWidth={2} />
            </LineChart>
          </ResponsiveContainer>
        )}
      </ChartShell>
    </div>
  );
}

function epochComponentSeries(rows: AttestationRewardRow[]) {
  const m = new Map<number, { head: number; source: number; target: number }>();
  for (const r of rows) {
    const cur = m.get(r.epoch) ?? { head: 0, source: 0, target: 0 };
    cur.head += r.head_reward;
    cur.source += r.source_reward;
    cur.target += r.target_reward;
    m.set(r.epoch, cur);
  }
  return [...m.entries()]
    .sort((a, b) => a[0] - b[0])
    .slice(-80)
    .map(([epoch, v]) => ({ epoch, head: v.head, source: v.source, target: v.target }));
}

function ChartShell({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <Card className="glass-panel border-border/80 bg-card/70">
      <CardHeader className="pb-1 pt-4 lg:pt-5">
        <CardTitle className="font-heading text-sm font-semibold tracking-tight lg:text-base">{title}</CardTitle>
        <CardDescription className="text-xs leading-relaxed lg:text-sm">{description}</CardDescription>
      </CardHeader>
      <CardContent className="h-[200px] pb-4 lg:h-[240px] xl:h-[260px]">{children}</CardContent>
    </Card>
  );
}

function EmptyChart() {
  return (
    <div className="flex h-full items-center justify-center text-xs text-muted-foreground">
      No rows for the current filters.
    </div>
  );
}

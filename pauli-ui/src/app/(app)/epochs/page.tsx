"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
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
import { fetchAllAttestationRewards } from "@/lib/api/attestation-rewards";
import { rewardsByEpochForValidator, aggregateTotalRewardByEpoch } from "@/lib/charts/aggregate-attestation";
import { formatInteger } from "@/lib/format";

type Applied = { from: number; to: number; validator?: number };

export default function EpochsPage() {
  const [fromEpoch, setFromEpoch] = useState("0");
  const [toEpoch, setToEpoch] = useState("500000");
  const [validator, setValidator] = useState("");

  const [applied, setApplied] = useState<Applied>({ from: 0, to: 500_000, validator: undefined });

  const applyFilters = () => {
    const from = Math.max(0, Number(fromEpoch) || 0);
    const to = Math.max(from, Number(toEpoch) || 0);
    const v = validator.trim();
    const validatorNum = v === "" ? undefined : Number(v);
    setApplied({
      from,
      to,
      validator: Number.isFinite(validatorNum) ? validatorNum : undefined,
    });
  };

  const q = useQuery({
    queryKey: ["epochs-chart", applied.from, applied.to, applied.validator ?? "all"],
    queryFn: async () => {
      const rows = await fetchAllAttestationRewards(
        {
          fromEpoch: applied.from,
          toEpoch: applied.to,
          maxRows: 40_000,
        },
        applied.validator,
      );
      const chart =
        applied.validator != null
          ? rewardsByEpochForValidator(rows).map((r) => ({
              epoch: r.epoch,
              totalGwei: r.totalGwei,
              head: r.head,
              source: r.source,
              target: r.target,
            }))
          : aggregateTotalRewardByEpoch(rows).map((r) => ({
              epoch: r.epoch,
              totalGwei: r.totalGwei,
              head: null as number | null,
              source: null as number | null,
              target: null as number | null,
            }));
      return { chart, rowCount: rows.length };
    },
  });

  if (q.isError) return <ApiErrorAlert error={q.error} />;

  return (
    <div className="space-y-8">
      <div>
        <h1 className="font-heading text-3xl font-bold tracking-tight">Epoch performance</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Attestation rewards by epoch. Optionally scope to one validator index.
        </p>
      </div>

      <div className="glass-panel flex flex-wrap items-end gap-4 rounded-xl border border-border/80 p-4">
        <div className="space-y-2">
          <Label htmlFor="from">from_epoch</Label>
          <Input
            id="from"
            className="w-40 font-mono"
            value={fromEpoch}
            onChange={(e) => setFromEpoch(e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="to">to_epoch</Label>
          <Input
            id="to"
            className="w-40 font-mono"
            value={toEpoch}
            onChange={(e) => setToEpoch(e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="val">validator_index (optional)</Label>
          <Input
            id="val"
            className="w-48 font-mono"
            placeholder="All validators"
            value={validator}
            onChange={(e) => setValidator(e.target.value)}
          />
        </div>
        <Button type="button" onClick={applyFilters} disabled={q.isFetching}>
          {q.isFetching ? "Loading…" : "Apply"}
        </Button>
      </div>

      {q.isPending ? (
        <Skeleton className="h-[320px] w-full" />
      ) : (
        <>
          <p className="text-xs text-muted-foreground">
            Loaded {formatInteger(q.data.rowCount)} reward rows (capped fetch). Chart points:{" "}
            {formatInteger(q.data.chart.length)}.
          </p>
          <ChartCard
            title={applied.validator != null ? "Validator rewards by epoch" : "Σ rewards by epoch"}
            description="Gwei values from attestation reward rows."
          >
            {q.data.chart.length === 0 ? (
              <p className="flex h-full items-center justify-center text-sm text-muted-foreground">
                No data for this window.
              </p>
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={q.data.chart} margin={{ top: 8, right: 16, left: 0, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke={chartTheme.grid} opacity={0.5} />
                  <XAxis dataKey="epoch" tick={{ fill: chartTheme.axis, fontSize: 11 }} />
                  <YAxis
                    tick={{ fill: chartTheme.axis, fontSize: 11 }}
                    tickFormatter={(v) => formatInteger(Number(v))}
                    width={80}
                  />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: chartTheme.tooltipBg,
                      border: `1px solid ${chartTheme.tooltipBorder}`,
                      borderRadius: 8,
                    }}
                  />
                  <Legend />
                  <Line
                    type="monotone"
                    dataKey="totalGwei"
                    name="total_reward"
                    stroke={chartTheme.primary}
                    dot={false}
                    strokeWidth={2}
                  />
                  {applied.validator != null ? (
                    <>
                      <Line
                        type="monotone"
                        dataKey="head"
                        name="head"
                        stroke={chartTheme.secondary}
                        dot={false}
                        strokeWidth={1.5}
                      />
                      <Line
                        type="monotone"
                        dataKey="source"
                        name="source"
                        stroke={chartTheme.tertiary}
                        dot={false}
                        strokeWidth={1.5}
                      />
                      <Line
                        type="monotone"
                        dataKey="target"
                        name="target"
                        stroke={chartTheme.axis}
                        dot={false}
                        strokeWidth={1.5}
                      />
                    </>
                  ) : null}
                </LineChart>
              </ResponsiveContainer>
            )}
          </ChartCard>
        </>
      )}
    </div>
  );
}

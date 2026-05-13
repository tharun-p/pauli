"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { cn } from "@/lib/utils";
import { useParams } from "next/navigation";
import { ApiErrorAlert } from "@/components/data/ApiErrorAlert";
import { StatusBadge } from "@/components/validators/StatusBadge";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
import { countSnapshots, getLatestSnapshot } from "@/lib/api/snapshots";
import { fetchAllAttestationRewards } from "@/lib/api/attestation-rewards";
import { fetchAllPenalties } from "@/lib/api/penalties";
import { formatGweiEth, formatInteger } from "@/lib/format";

export default function ValidatorDetailPage() {
  const params = useParams();
  const validatorIndex = String(params.validatorIndex ?? "");

  const overview = useQuery({
    queryKey: ["validator-detail", validatorIndex],
    queryFn: async () => {
      const [latest, count] = await Promise.all([
        getLatestSnapshot(validatorIndex),
        countSnapshots(validatorIndex),
      ]);
      return { latest, count };
    },
    enabled: validatorIndex.length > 0,
  });

  const rewards = useQuery({
    queryKey: ["validator-detail-rewards", validatorIndex],
    queryFn: () =>
      fetchAllAttestationRewards(
        { fromEpoch: 0, toEpoch: 2_000_000, maxRows: 5000 },
        validatorIndex,
      ),
    enabled: validatorIndex.length > 0,
  });

  const penalties = useQuery({
    queryKey: ["validator-detail-penalties", validatorIndex],
    queryFn: () =>
      fetchAllPenalties(validatorIndex, { fromEpoch: 0, toEpoch: 2_000_000 }, 3000),
    enabled: validatorIndex.length > 0,
  });

  if (overview.isError) return <ApiErrorAlert error={overview.error} />;

  return (
    <div className="space-y-8">
      <div className="flex flex-wrap items-center gap-4">
        <Link
          href="/validators"
          className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "inline-flex")}
        >
          ← Back to list
        </Link>
        <h1 className="font-heading text-3xl font-bold tracking-tight">
          Validator <span className="font-mono text-primary">{validatorIndex}</span>
        </h1>
      </div>

      {overview.isPending ? (
        <Skeleton className="h-40 w-full max-w-xl" />
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          <Card className="glass-panel border-border/80 bg-card/80">
            <CardHeader>
              <CardTitle className="font-heading text-lg">Latest snapshot</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              {overview.data.latest ? (
                <>
                  <div className="flex justify-between gap-4">
                    <span className="text-muted-foreground">Status</span>
                    <StatusBadge status={overview.data.latest.status} />
                  </div>
                  <div className="flex justify-between font-mono">
                    <span className="text-muted-foreground">Slot</span>
                    {formatInteger(overview.data.latest.slot)}
                  </div>
                  <div className="flex justify-between font-mono">
                    <span className="text-muted-foreground">Balance</span>
                    {formatGweiEth(overview.data.latest.balance)}
                  </div>
                  <div className="flex justify-between font-mono">
                    <span className="text-muted-foreground">Effective</span>
                    {formatGweiEth(overview.data.latest.effective_balance)}
                  </div>
                  <div className="flex justify-between text-xs text-muted-foreground">
                    <span>Timestamp</span>
                    <span>{overview.data.latest.timestamp}</span>
                  </div>
                </>
              ) : (
                <p className="text-muted-foreground">No snapshot rows for this validator yet.</p>
              )}
            </CardContent>
          </Card>
          <Card className="glass-panel border-border/80 bg-card/80">
            <CardHeader>
              <CardTitle className="font-heading text-lg">Storage</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="font-heading text-4xl font-bold text-primary">
                {formatInteger(overview.data.count)}
              </p>
              <p className="mt-2 text-sm text-muted-foreground">Snapshot rows stored</p>
            </CardContent>
          </Card>
        </div>
      )}

      <Tabs defaultValue="rewards" className="space-y-4">
        <TabsList className="bg-muted/40">
          <TabsTrigger value="rewards">Attestation rewards</TabsTrigger>
          <TabsTrigger value="penalties">Penalties</TabsTrigger>
        </TabsList>
        <TabsContent value="rewards">
          {rewards.isError ? (
            <ApiErrorAlert error={rewards.error} />
          ) : rewards.isPending ? (
            <Skeleton className="h-48 w-full" />
          ) : (
            <div className="glass-panel max-h-[400px] overflow-auto rounded-xl border border-border/80">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Epoch</TableHead>
                    <TableHead>Total</TableHead>
                    <TableHead>Head</TableHead>
                    <TableHead>Source</TableHead>
                    <TableHead>Target</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rewards.data.slice(0, 400).map((r) => (
                    <TableRow key={`${r.epoch}-${r.timestamp}`}>
                      <TableCell className="font-mono">{formatInteger(r.epoch)}</TableCell>
                      <TableCell className="font-mono">{formatInteger(r.total_reward)}</TableCell>
                      <TableCell className="font-mono">{formatInteger(r.head_reward)}</TableCell>
                      <TableCell className="font-mono">{formatInteger(r.source_reward)}</TableCell>
                      <TableCell className="font-mono">{formatInteger(r.target_reward)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </TabsContent>
        <TabsContent value="penalties">
          {penalties.isError ? (
            <ApiErrorAlert error={penalties.error} />
          ) : penalties.isPending ? (
            <Skeleton className="h-48 w-full" />
          ) : penalties.data.length === 0 ? (
            <p className="text-sm text-muted-foreground">No penalty rows in range.</p>
          ) : (
            <div className="glass-panel max-h-[400px] overflow-auto rounded-xl border border-border/80">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Epoch</TableHead>
                    <TableHead>Slot</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Gwei</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {penalties.data.map((p) => (
                    <TableRow key={`${p.epoch}-${p.slot}-${p.penalty_type}`}>
                      <TableCell className="font-mono">{formatInteger(p.epoch)}</TableCell>
                      <TableCell className="font-mono">{formatInteger(p.slot)}</TableCell>
                      <TableCell className="capitalize">{p.penalty_type.replaceAll("_", " ")}</TableCell>
                      <TableCell className="font-mono">{formatInteger(p.penalty_gwei)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}

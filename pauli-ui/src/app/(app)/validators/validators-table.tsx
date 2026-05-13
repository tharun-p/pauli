"use client";

import { useQuery, useQueries } from "@tanstack/react-query";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { ApiErrorAlert } from "@/components/data/ApiErrorAlert";
import { StatusBadge } from "@/components/validators/StatusBadge";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { listValidators } from "@/lib/api/validators";
import { countSnapshots, getLatestSnapshot } from "@/lib/api/snapshots";
import { formatGweiEth, formatInteger } from "@/lib/format";
import { cn } from "@/lib/utils";

const PAGE_SIZE = 25;

export function ValidatorsTable() {
  const search = useSearchParams();
  const offset = Math.max(0, Number(search.get("offset") ?? "0") || 0);

  const listQ = useQuery({
    queryKey: ["validators", PAGE_SIZE, offset],
    queryFn: () => listValidators(PAGE_SIZE, offset),
  });

  const indices = listQ.data?.data.map((r) => r.validator_index) ?? [];

  const summaries = useQueries({
    queries: indices.map((validatorIndex) => ({
      queryKey: ["validator-summary", validatorIndex],
      queryFn: async () => {
        const [latest, count] = await Promise.all([
          getLatestSnapshot(validatorIndex),
          countSnapshots(validatorIndex),
        ]);
        return { validatorIndex, latest, count };
      },
      enabled: listQ.isSuccess && indices.length > 0,
    })),
  });

  const loadingSummaries = summaries.some((s) => s.isPending);

  if (listQ.isError) return <ApiErrorAlert error={listQ.error} />;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-heading text-3xl font-bold tracking-tight">Validator monitor</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Indices with snapshot rows. Open a row for detail.
        </p>
      </div>

      <div className="glass-panel overflow-hidden rounded-xl border border-border/80">
        {!listQ.isSuccess ? (
          <div className="space-y-2 p-6">
            {[1, 2, 3, 4, 5].map((i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="border-border/60 hover:bg-transparent">
                <TableHead className="w-[140px]">Validator</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Latest slot</TableHead>
                <TableHead>Balance</TableHead>
                <TableHead>Effective</TableHead>
                <TableHead className="text-right">Snapshots</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {listQ.data.data.map((row, i) => {
                const sum = summaries[i]?.data;
                const pending = summaries[i]?.isPending;
                return (
                  <TableRow key={row.validator_index} className="border-border/40">
                    <TableCell>
                      <Link
                        href={`/validators/${row.validator_index}`}
                        className="font-mono text-primary hover:underline"
                      >
                        {formatInteger(row.validator_index)}
                      </Link>
                    </TableCell>
                    <TableCell>
                      {pending ? (
                        <Skeleton className="h-5 w-24" />
                      ) : sum?.latest ? (
                        <StatusBadge status={sum.latest.status} />
                      ) : (
                        <span className="text-sm text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell className="font-mono text-sm">
                      {pending ? (
                        <Skeleton className="h-4 w-16" />
                      ) : sum?.latest ? (
                        formatInteger(sum.latest.slot)
                      ) : (
                        "—"
                      )}
                    </TableCell>
                    <TableCell className="font-mono text-sm">
                      {pending ? (
                        <Skeleton className="h-4 w-20" />
                      ) : sum?.latest ? (
                        formatGweiEth(sum.latest.balance)
                      ) : (
                        "—"
                      )}
                    </TableCell>
                    <TableCell className="font-mono text-sm">
                      {pending ? (
                        <Skeleton className="h-4 w-20" />
                      ) : sum?.latest ? (
                        formatGweiEth(sum.latest.effective_balance)
                      ) : (
                        "—"
                      )}
                    </TableCell>
                    <TableCell className="text-right font-mono text-sm">
                      {pending ? <Skeleton className="ml-auto h-4 w-10" /> : formatInteger(sum?.count ?? 0)}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </div>

      <div className="flex flex-wrap items-center justify-between gap-4">
        <p className="text-xs text-muted-foreground">
          Page size {PAGE_SIZE}. Showing {listQ.data?.meta.count ?? 0} rows (offset {offset}).
        </p>
        <div className="flex gap-2">
          {offset === 0 ? (
            <Button variant="outline" size="sm" disabled>
              Previous
            </Button>
          ) : (
            <Link
              href={`/validators?offset=${Math.max(0, offset - PAGE_SIZE)}`}
              className={cn(buttonVariants({ variant: "outline", size: "sm" }), "inline-flex")}
            >
              Previous
            </Link>
          )}
          {listQ.isPending || loadingSummaries || !listQ.isSuccess || listQ.data.data.length < PAGE_SIZE ? (
            <Button variant="outline" size="sm" disabled>
              Next
            </Button>
          ) : (
            <Link
              href={`/validators?offset=${offset + PAGE_SIZE}`}
              className={cn(buttonVariants({ variant: "outline", size: "sm" }), "inline-flex")}
            >
              Next
            </Link>
          )}
        </div>
      </div>
    </div>
  );
}

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

function fieldLabel(className?: string) {
  return cn(
    "text-[11px] font-medium uppercase tracking-wide text-muted-foreground",
    className,
  );
}

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

      <div className="glass-panel overflow-hidden rounded-xl border border-border/80 px-4 py-3 sm:px-6 sm:py-4">
        {!listQ.isSuccess ? (
          <div className="space-y-2 py-1">
            {[1, 2, 3, 4, 5].map((i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : (
          <>
            <ul className="md:hidden space-y-3">
              {listQ.data.data.map((row, i) => {
                const sum = summaries[i]?.data;
                const pending = summaries[i]?.isPending;
                return (
                  <li
                    key={row.validator_index}
                    className="rounded-lg border border-border/60 bg-card/20 p-4 shadow-sm"
                  >
                    <div className="flex gap-4">
                      <div className="min-w-0 flex-1 space-y-4">
                        <div className="min-w-0">
                          <p className={fieldLabel()}>Validator</p>
                          <Link
                            href={`/validators/${row.validator_index}`}
                            className="mt-1 inline-block font-mono text-base text-primary hover:underline"
                          >
                            {formatInteger(row.validator_index)}
                          </Link>
                        </div>
                        <div className="min-w-0">
                          <p className={fieldLabel()}>Status</p>
                          <div className="mt-1 flex flex-wrap gap-2">
                            {pending ? (
                              <Skeleton className="h-6 w-32 max-w-full" />
                            ) : sum?.latest ? (
                              <StatusBadge status={sum.latest.status} />
                            ) : (
                              <span className="text-sm text-muted-foreground">—</span>
                            )}
                          </div>
                        </div>
                        <div className="min-w-0">
                          <p className={fieldLabel()}>Latest slot</p>
                          <p className="mt-1 font-mono text-sm break-words text-foreground">
                            {pending ? (
                              <Skeleton className="mt-1 inline-block h-4 w-16" />
                            ) : sum?.latest ? (
                              formatInteger(sum.latest.slot)
                            ) : (
                              "—"
                            )}
                          </p>
                        </div>
                      </div>

                      <div className="flex w-[9.5rem] shrink-0 flex-col justify-between gap-4 self-stretch text-right text-sm sm:w-[10.5rem]">
                        <div className="min-w-0">
                          <p className={fieldLabel("text-right")}>Balance</p>
                          <p className="mt-1 font-mono break-words text-foreground">
                            {pending ? (
                              <Skeleton className="ml-auto mt-1 h-4 w-24 max-w-full" />
                            ) : sum?.latest ? (
                              formatGweiEth(sum.latest.balance)
                            ) : (
                              "—"
                            )}
                          </p>
                        </div>
                        <div className="min-w-0">
                          <p className={fieldLabel("text-right")}>Effective</p>
                          <p className="mt-1 font-mono break-words text-foreground">
                            {pending ? (
                              <Skeleton className="ml-auto mt-1 h-4 w-24 max-w-full" />
                            ) : sum?.latest ? (
                              formatGweiEth(sum.latest.effective_balance)
                            ) : (
                              "—"
                            )}
                          </p>
                        </div>
                      </div>
                    </div>
                  </li>
                );
              })}
            </ul>

            <div className="hidden md:block">
              <Table className="text-sm" containerClassName="w-full overflow-x-auto">
                <TableHeader>
                  <TableRow className="border-border/60 hover:bg-transparent [&_th]:px-4 [&_th]:py-3">
                    <TableHead className="w-[140px] whitespace-nowrap">Validator</TableHead>
                    <TableHead className="whitespace-nowrap">Status</TableHead>
                    <TableHead className="whitespace-nowrap">Latest slot</TableHead>
                    <TableHead className="whitespace-nowrap">Balance</TableHead>
                    <TableHead className="whitespace-nowrap">Effective</TableHead>
                    <TableHead className="text-right whitespace-nowrap">Snapshots</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {listQ.data.data.map((row, i) => {
                    const sum = summaries[i]?.data;
                    const pending = summaries[i]?.isPending;
                    return (
                      <TableRow key={row.validator_index} className="border-border/40 [&_td]:px-4 [&_td]:py-3">
                        <TableCell className="font-mono">
                          <Link
                            href={`/validators/${row.validator_index}`}
                            className="text-primary hover:underline"
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
                            <span className="text-muted-foreground">—</span>
                          )}
                        </TableCell>
                        <TableCell className="font-mono">
                          {pending ? (
                            <Skeleton className="h-4 w-16" />
                          ) : sum?.latest ? (
                            formatInteger(sum.latest.slot)
                          ) : (
                            "—"
                          )}
                        </TableCell>
                        <TableCell className="font-mono">
                          {pending ? (
                            <Skeleton className="h-4 w-20" />
                          ) : sum?.latest ? (
                            formatGweiEth(sum.latest.balance)
                          ) : (
                            "—"
                          )}
                        </TableCell>
                        <TableCell className="font-mono">
                          {pending ? (
                            <Skeleton className="h-4 w-20" />
                          ) : sum?.latest ? (
                            formatGweiEth(sum.latest.effective_balance)
                          ) : (
                            "—"
                          )}
                        </TableCell>
                        <TableCell className="text-right font-mono">
                          {pending ? <Skeleton className="ml-auto h-4 w-10" /> : formatInteger(sum?.count ?? 0)}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          </>
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

"use client";

import { Suspense } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { ValidatorsTable } from "./validators-table";

export default function ValidatorsPage() {
  return (
    <Suspense
      fallback={
        <div className="space-y-6">
          <Skeleton className="h-10 w-64" />
          <Skeleton className="h-64 w-full" />
        </div>
      }
    >
      <ValidatorsTable />
    </Suspense>
  );
}

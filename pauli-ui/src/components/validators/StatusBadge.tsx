import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

const statusTone: Record<string, "default" | "secondary" | "outline" | "destructive"> = {
  active_ongoing: "default",
  active_exiting: "secondary",
  active_slashed: "destructive",
  pending_initialized: "outline",
  pending_queued: "outline",
  exited_unslashed: "secondary",
  exited_slashed: "destructive",
  withdrawal_possible: "secondary",
  withdrawal_done: "outline",
};

export function StatusBadge({ status }: { status: string }) {
  const variant = statusTone[status] ?? "outline";
  return (
    <Badge
      variant={variant}
      className={cn(
        "font-mono text-xs font-normal capitalize",
        variant === "default" && "border-primary/40 bg-primary/15 text-primary",
      )}
    >
      {status.replaceAll("_", " ")}
    </Badge>
  );
}

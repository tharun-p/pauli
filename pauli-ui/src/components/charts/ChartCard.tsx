import type { ReactNode } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { cn } from "@/lib/utils";

type ChartCardProps = {
  title: string;
  description?: string;
  children: ReactNode;
  className?: string;
};

export function ChartCard({ title, description, children, className }: ChartCardProps) {
  return (
    <Card className={cn("glass-panel border-border/80 bg-card/80", className)}>
      <CardHeader className="pb-2">
        <CardTitle className="font-heading text-lg tracking-tight">{title}</CardTitle>
        {description ? (
          <CardDescription className="text-muted-foreground">{description}</CardDescription>
        ) : null}
      </CardHeader>
      <CardContent className="h-[280px] pt-0">{children}</CardContent>
    </Card>
  );
}

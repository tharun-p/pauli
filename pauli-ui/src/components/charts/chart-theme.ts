import { themePalette } from "@/styles/stitch-palette";

/** Recharts strokes aligned with dashboard reference (lime + neutrals). */
export const chartTheme = {
  axis: themePalette.onSurfaceVariant,
  grid: themePalette.outlineVariant,
  primary: themePalette.primary,
  secondary: "#ffffff",
  tertiary: themePalette.onSurfaceVariant,
  tooltipBg: themePalette.surfaceContainerHigh,
  tooltipBorder: themePalette.outline,
} as const;

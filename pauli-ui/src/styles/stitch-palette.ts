/**
 * Dashboard reference: high-contrast dark UI — near-black surfaces, chartreuse (#DFFF00) accent,
 * white primary text, muted grey labels (~#A0A0A0). Used by Recharts and hard-coded chart strokes.
 */
export const themePalette = {
  background: "#000000",
  surface: "#000000",
  surfaceContainer: "#121212",
  surfaceContainerHigh: "#1a1a1a",
  surfaceContainerHighest: "#242424",
  onSurface: "#ffffff",
  onSurfaceVariant: "#a0a0a0",
  primary: "#dfff00",
  primaryDim: "#b8cf00",
  primaryContainer: "#e8ff4d",
  onPrimary: "#000000",
  secondary: "#2a2a2a",
  secondaryDim: "#3d3d3d",
  tertiary: "#737373",
  outlineVariant: "#2a2a2a",
  outline: "#3d3d3d",
  error: "#f87171",
} as const;

/** @deprecated use themePalette — kept for gradual refactors */
export const stitchPalette = themePalette;

export const chartColors = {
  primary: themePalette.primary,
  secondary: "#ffffff",
  tertiary: themePalette.onSurfaceVariant,
  muted: "#525252",
} as const;

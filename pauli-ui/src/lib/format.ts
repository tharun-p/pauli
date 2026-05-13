const GWEI = 1e9;

export function formatGweiEth(gwei: number, fractionDigits = 4): string {
  const eth = gwei / GWEI;
  if (!Number.isFinite(eth)) return "—";
  return `${eth.toFixed(fractionDigits)} ETH`;
}

export function formatInteger(n: number): string {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(n);
}

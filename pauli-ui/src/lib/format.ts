const GWEI = 1e9;
const WEI_PER_ETH = BigInt(10) ** BigInt(18);

export function formatGweiEth(gwei: number, fractionDigits = 4): string {
  const eth = gwei / GWEI;
  if (!Number.isFinite(eth)) return "—";
  return `${eth.toFixed(fractionDigits)} ETH`;
}

/** Display a decimal wei string as ETH (BigInt-safe for typical block tips). */
export function formatWeiEth(weiStr: string | null | undefined, fractionDigits = 6): string {
  if (weiStr == null || weiStr === "") return "—";
  try {
    const wei = BigInt(weiStr);
    if (wei === BigInt(0)) return `0.${"0".repeat(fractionDigits)} ETH`;
    const neg = wei < BigInt(0);
    const w = neg ? -wei : wei;
    const whole = w / WEI_PER_ETH;
    const rem = w % WEI_PER_ETH;
    const scale = BigInt(10) ** BigInt(fractionDigits);
    const frac = (rem * scale) / WEI_PER_ETH;
    const fracStr = frac.toString().padStart(fractionDigits, "0");
    return `${neg ? "-" : ""}${whole.toString()}.${fracStr} ETH`;
  } catch {
    return "—";
  }
}

export function formatInteger(n: number): string {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(n);
}

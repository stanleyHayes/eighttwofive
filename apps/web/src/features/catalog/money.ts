import type { SizeBand } from "./api";

/** 50000 -> "GH₵ 500.00" */
export function formatPesewas(pesewas: number): string {
  const ghs = pesewas / 100;
  return `GH₵ ${ghs.toLocaleString("en-US", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}`;
}

/** 50000 -> "500.00" (for editing in a text field). */
export function pesewasToGhsInput(pesewas: number): string {
  return (pesewas / 100).toFixed(2);
}

/**
 * "500" | "500.5" | "1,250.00" -> integer pesewas. Returns null when the
 * value is not a valid GHS amount with at most two decimal places.
 */
export function ghsInputToPesewas(value: string): number | null {
  const trimmed = value.trim().replace(/,/g, "");
  if (!/^\d+(\.\d{1,2})?$/.test(trimmed)) return null;
  const pesewas = Math.round(Number(trimmed) * 100);
  return Number.isSafeInteger(pesewas) ? pesewas : null;
}

export function formatPriceRange(bands: SizeBand[]): string {
  if (bands.length === 0) return "—";
  const prices = bands.map((band) => band.pricePesewas);
  const min = Math.min(...prices);
  const max = Math.max(...prices);
  return min === max ? formatPesewas(min) : `${formatPesewas(min)} – ${formatPesewas(max)}`;
}

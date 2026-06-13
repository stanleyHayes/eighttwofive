import { ApiError, request } from "@/lib/api";

export interface TimeBucket {
  label: string;
  startAt: string;
  revenuePesewas: number;
  orderCount: number;
}

export interface DesignStat {
  designId: string;
  name: string;
  orderCount: number;
  revenuePesewas: number;
}

export interface CollectionStat {
  collectionId: string;
  name: string;
  orderCount: number;
  revenuePesewas: number;
}

export interface RecentOrder {
  ref: string;
  type: string;
  status: string;
  totalPesewas: number;
  createdAt: string;
}

export interface PeriodComparison {
  currentRevenuePesewas: number;
  priorRevenuePesewas: number;
  currentOrderCount: number;
  priorOrderCount: number;
  revenueChangeBps: number;
  orderCountChangeBps: number;
}

export interface StoreAnalytics {
  waitlistCount: number;
  customerCount: number;
  orderCount: number;
  bookedRevenuePesewas: number;
  averageOrderValuePesewas: number;
  ordersByStatus: Record<string, number>;
  ordersByType: Record<string, number>;
  revenuePesewas: number;
  collectionViews: number;
  comparison: PeriodComparison;
  revenueSeries: TimeBucket[];
  topDesigns: DesignStat[];
  topCollections: CollectionStat[];
  recentOrders: RecentOrder[];
}

export function errorMessage(
  error: unknown,
  fallback = "Something went wrong. Try again in a moment.",
): string {
  return error instanceof ApiError ? error.message : fallback;
}

const EMPTY_COMPARISON: PeriodComparison = {
  currentRevenuePesewas: 0,
  priorRevenuePesewas: 0,
  currentOrderCount: 0,
  priorOrderCount: 0,
  revenueChangeBps: 0,
  orderCountChangeBps: 0,
};

/**
 * Coerce the raw payload into a fully-populated shape. The Go API marshals
 * empty slices/maps as `null`, so every collection is defaulted here — the
 * dashboard can then read arrays without guarding each access.
 */
function normalize(raw: StoreAnalytics): StoreAnalytics {
  return {
    ...raw,
    ordersByStatus: raw.ordersByStatus ?? {},
    ordersByType: raw.ordersByType ?? {},
    comparison: raw.comparison ?? EMPTY_COMPARISON,
    revenueSeries: raw.revenueSeries ?? [],
    topDesigns: raw.topDesigns ?? [],
    topCollections: raw.topCollections ?? [],
    recentOrders: raw.recentOrders ?? [],
  };
}

export async function getStoreAnalytics(): Promise<StoreAnalytics> {
  return normalize(await request<StoreAnalytics>("/api/v1/admin/analytics"));
}

export function statusLabel(status: string): string {
  return status.replace(/_/g, " ");
}

export function typeLabel(type: string): string {
  switch (type) {
    case "standard":
      return "Standard";
    case "custom_size":
      return "Custom size";
    case "design_change":
      return "Design change";
    case "visit":
      return "Visit booking";
    default:
      return type;
  }
}

/**
 * Render integer basis points as a signed percent, e.g. 3000 -> "+30%" and
 * -2500 -> "−25%". Positives gain a leading "+"; negatives use a typographic
 * minus (U+2212) so the sign reads cleanly next to the figure.
 */
export function formatBps(bps: number): string {
  const rounded = Math.round((bps / 100) * 10) / 10;
  const magnitude = Math.abs(rounded).toLocaleString("en-GH", { maximumFractionDigits: 1 });

  if (rounded < 0) {
    return `−${magnitude}%`;
  }

  const sign = rounded > 0 ? "+" : "";

  return `${sign}${magnitude}%`;
}

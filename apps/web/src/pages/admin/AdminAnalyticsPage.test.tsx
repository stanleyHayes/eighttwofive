import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { AdminAnalyticsPage } from "./AdminAnalyticsPage";
import type { StoreAnalytics } from "@/features/analytics/api";

function bucket(label: string, revenuePesewas: number, orderCount: number) {
  return { label, startAt: `2026-0${label}`, revenuePesewas, orderCount };
}

const analytics: StoreAnalytics = {
  waitlistCount: 7,
  customerCount: 4,
  orderCount: 9,
  bookedRevenuePesewas: 900_000,
  averageOrderValuePesewas: 100_000,
  ordersByStatus: { booked: 2, requested: 1, in_production: 1 },
  ordersByType: { standard: 2, custom_size: 1, visit: 1 },
  revenuePesewas: 900_000,
  collectionViews: 0,
  comparison: {
    currentRevenuePesewas: 600_000,
    priorRevenuePesewas: 400_000,
    currentOrderCount: 6,
    priorOrderCount: 4,
    revenueChangeBps: 5_000,
    orderCountChangeBps: -2_500,
  },
  revenueSeries: [
    bucket("1", 0, 0),
    bucket("2", 100_000, 1),
    bucket("3", 200_000, 2),
    bucket("4", 0, 0),
    bucket("5", 50_000, 1),
    bucket("6", 0, 0),
    bucket("7", 150_000, 1),
    bucket("8", 0, 0),
    bucket("9", 100_000, 1),
    bucket("10", 0, 0),
    bucket("11", 100_000, 1),
    bucket("12", 200_000, 2),
  ],
  topDesigns: [
    { designId: "d1", name: "Atelier Blazer", orderCount: 5, revenuePesewas: 500_000 },
    { designId: "d2", name: "Studio Dress", orderCount: 3, revenuePesewas: 300_000 },
  ],
  topCollections: [
    { collectionId: "c1", name: "Heritage", orderCount: 6, revenuePesewas: 600_000 },
    { collectionId: "c2", name: "Studio", orderCount: 3, revenuePesewas: 300_000 },
  ],
  recentOrders: [
    { ref: "E25-9", type: "standard", status: "booked", totalPesewas: 100_000, createdAt: "2026-06-10T10:00:00Z" },
    { ref: "E25-8", type: "visit", status: "requested", totalPesewas: 0, createdAt: "2026-06-09T10:00:00Z" },
  ],
};

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

function mockFetch(payload: unknown = { data: analytics }) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";

    if (url === "/api/v1/admin/analytics" && method === "GET") {
      return jsonResponse(200, payload);
    }

    return jsonResponse(404, { error: { code: "not_found", message: "no route" } });
  });
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AdminAnalyticsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AdminAnalyticsPage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders headline stat cards from analytics data", async () => {
    vi.stubGlobal("fetch", mockFetch());
    renderPage();

    expect(await screen.findByRole("heading", { name: /analytics/i })).toBeInTheDocument();

    expect(await screen.findByTestId("revenue-card")).toHaveTextContent("GH₵ 9,000.00");
    expect(screen.getByTestId("orders-card")).toHaveTextContent("9");
    expect(screen.getByTestId("aov-card")).toHaveTextContent("GH₵ 1,000.00");
    expect(screen.getByTestId("waitlist-card")).toHaveTextContent("7");
    expect(screen.getByTestId("customers-card")).toHaveTextContent("4");
  });

  it("renders signed percent deltas on the headline cards", async () => {
    vi.stubGlobal("fetch", mockFetch());
    renderPage();

    expect(await screen.findByTestId("revenue-card")).toHaveTextContent("+50%");
    expect(screen.getByTestId("orders-card")).toHaveTextContent("−25%");
  });

  it("draws the 12-bucket revenue chart with bars", async () => {
    vi.stubGlobal("fetch", mockFetch());
    renderPage();

    const chart = await screen.findByTestId("revenue-chart");
    expect(chart).toBeInTheDocument();
    expect(screen.getAllByTestId("revenue-bar")).toHaveLength(12);
  });

  it("renders top designs and top collections tables", async () => {
    vi.stubGlobal("fetch", mockFetch());
    renderPage();

    const designs = await screen.findByTestId("top-designs");
    expect(designs).toHaveTextContent("Atelier Blazer");
    expect(designs).toHaveTextContent("Studio Dress");

    const collections = screen.getByTestId("top-collections");
    expect(collections).toHaveTextContent("Heritage");
    expect(collections).toHaveTextContent("GH₵ 6,000.00");
  });

  it("renders the recent-orders activity list", async () => {
    vi.stubGlobal("fetch", mockFetch());
    renderPage();

    const recent = await screen.findByTestId("recent-orders");
    expect(recent).toHaveTextContent("E25-9");
    expect(recent).toHaveTextContent("Visit booking");
  });

  it("shows a calm empty state when there are no orders", async () => {
    vi.stubGlobal(
      "fetch",
      mockFetch({
        data: {
          ...analytics,
          orderCount: 0,
          recentOrders: [],
        },
      }),
    );

    renderPage();

    expect(await screen.findByTestId("analytics-empty")).toHaveTextContent("No orders yet");
    expect(screen.queryByTestId("analytics-dashboard")).not.toBeInTheDocument();
  });
});

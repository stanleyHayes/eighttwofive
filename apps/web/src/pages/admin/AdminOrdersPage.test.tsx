import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { AdminOrdersPage } from "./AdminOrdersPage";
import type { Order } from "@/features/orders/api";

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

function makeOrder(overrides: Partial<Order> & { ref: string }): Order {
  return {
    id: overrides.id ?? overrides.ref,
    ref: overrides.ref,
    customerId: "u1",
    designId: "d1",
    designSnapshot: {
      name: overrides.designSnapshot?.name ?? "Midnight Blazer",
      photoPublicId: "photos/midnight",
      pricePesewas: 125_000,
    },
    type: overrides.type ?? "standard",
    customisation: overrides.customisation ?? { sizeMode: "band", bandLabel: "12" },
    quote: overrides.quote ?? { pricePesewas: 0, timeline: "", notes: "" },
    delivery: overrides.delivery ?? { mode: "pickup" },
    payments: overrides.payments ?? [],
    status: overrides.status ?? "booked",
    statusHistory: overrides.statusHistory ?? [
      { status: overrides.status ?? "booked", at: "2026-06-10T10:00:00Z", by: "system" },
    ],
    customerPhone: overrides.customerPhone ?? "+233241234567",
    totalPesewas: overrides.totalPesewas ?? 125_000,
    createdAt: "2026-06-10T10:00:00Z",
    updatedAt: "2026-06-10T10:00:00Z",
  };
}

const orders: Order[] = [
  makeOrder({ ref: "E25-0001", type: "standard", status: "booked" }),
  makeOrder({
    ref: "E25-0002",
    type: "custom_size",
    status: "requested",
    designSnapshot: { name: "Bespoke Trouser", photoPublicId: "photos/trouser", pricePesewas: 0 },
    customisation: { sizeMode: "custom", measurements: { waist: "32" } },
  }),
];

function mockFetch() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";

    if (url.includes("/api/v1/admin/orders") && method === "GET" && !url.includes("/api/v1/admin/orders/")) {
      return jsonResponse(200, {
        data: { items: orders, total: orders.length, page: 1, pageSize: 20 },
      });
    }

    const match = url.match(/\/api\/v1\/admin\/orders\/([^/]+)$/);
    if (match && method === "GET") {
      const order = orders.find((o) => o.ref === match[1]);
      return order ? jsonResponse(200, { data: order }) : jsonResponse(404, { error: { code: "not_found", message: "no order" } });
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
        <AdminOrdersPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AdminOrdersPage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders order buckets and opens the detail panel when a row is selected", async () => {
    vi.stubGlobal("fetch", mockFetch());
    renderPage();

    expect(await screen.findByRole("heading", { name: /orders/i })).toBeInTheDocument();
    expect(await screen.findByText("E25-0001")).toBeInTheDocument();
    expect(screen.getByText("E25-0002")).toBeInTheDocument();

    await userEvent.click(screen.getByText("E25-0002"));

    expect(await screen.findByRole("heading", { name: "Bespoke Trouser" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /mark paid manually/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /send payment link/i })).toBeInTheDocument();
  });
});

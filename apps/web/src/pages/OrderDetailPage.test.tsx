import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router";
import { OrderDetailPage } from "./OrderDetailPage";
import type { Order, OrderStatus } from "@/features/orders/api";

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

const defaultSettings = {
  depositPesewas: 500_00,
  whatsappNumber: "+233200000000",
  visitLocation: "Osu, Accra",
  cloudName: "demo",
  deliveryRates: [{ area: "Accra", ratePesewas: 1000 }],
};

function makeOrder(status: OrderStatus, timeline = "2 weeks"): Order {
  return {
    id: "o1",
    ref: "E25-0001",
    customerId: "u1",
    designId: "d1",
    designSnapshot: {
      name: "Midnight Blazer",
      photoPublicId: "photos/midnight",
      pricePesewas: 125_000,
    },
    type: "standard",
    customisation: { sizeMode: "band", bandLabel: "12" },
    quote: { pricePesewas: 0, timeline, notes: "" },
    delivery: { mode: "pickup" },
    payments: [{ providerRef: "pi_1", amountPesewas: 125_000, status: "success", method: "mobile_money" }],
    status,
    statusHistory: [{ status, at: "2026-06-10T10:00:00Z", by: "payment_webhook" }],
    customerPhone: "+233241234567",
    totalPesewas: 125_000,
    createdAt: "2026-06-10T10:00:00Z",
    updatedAt: "2026-06-10T10:00:00Z",
  };
}

function mockFetch(order: Order) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";

    if (url.includes("/api/v1/settings") && method === "GET") {
      return jsonResponse(200, { data: defaultSettings });
    }
    if (url.includes(`/api/v1/orders/${order.ref}`) && method === "GET") {
      return jsonResponse(200, { data: order });
    }

    return jsonResponse(404, { error: { code: "not_found", message: "no route" } });
  });
}

function renderPage(ref: string) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/account/orders/${ref}`]}>
        <Routes>
          <Route path="/account/orders/:ref" element={<OrderDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("OrderDetailPage stage rendering", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it.each([
    ["booked", "order confirmed"],
    ["in_production", "in production"],
    ["ready", "ready"],
    ["fulfilled", "ready"],
  ] as const)("maps status %s to customer-facing stage %s", async (status, expected) => {
    const order = makeOrder(status);
    vi.stubGlobal("fetch", mockFetch(order));
    renderPage(order.ref);

    expect(await screen.findByRole("heading", { name: expected, level: 1 })).toBeInTheDocument();
    expect(screen.getByText(`order ${order.ref}`)).toBeInTheDocument();
  });

  it("shows the status alert with timeframe and email indicator for booked, in_production, and ready", async () => {
    const order = makeOrder("in_production", "10 working days");
    vi.stubGlobal("fetch", mockFetch(order));
    renderPage(order.ref);

    expect(await screen.findByText("Estimated: 10 working days")).toBeInTheDocument();
    expect(screen.getByText(/we'll email you at each step/i)).toBeInTheDocument();
  });

  it("falls back to the generic timeframe when no quote timeline is set", async () => {
    const order = makeOrder("booked", "");
    vi.stubGlobal("fetch", mockFetch(order));
    renderPage(order.ref);

    expect(
      await screen.findByText(/roughly two weeks, depending on current bookings/i),
    ).toBeInTheDocument();
  });

  it("does not show the status alert for non-customer-facing statuses", async () => {
    const order = makeOrder("pending_payment", "");
    vi.stubGlobal("fetch", mockFetch(order));
    renderPage(order.ref);

    expect(await screen.findByText("awaiting payment")).toBeInTheDocument();
    expect(screen.queryByText(/we'll email you at each step/i)).not.toBeInTheDocument();
  });

  it("shows the booked visit card and call button for visit orders", async () => {
    const order = makeOrder("booked");
    order.type = "visit";
    order.customisation.sizeMode = "home_visit";
    vi.stubGlobal("fetch", mockFetch(order));
    renderPage(order.ref);

    expect(await screen.findByText(/booked visit/i)).toBeInTheDocument();
    const callButton = screen.getByRole("link", { name: /call eight two five/i });
    expect(callButton).toHaveAttribute("href", `tel:${defaultSettings.whatsappNumber}`);
  });
});

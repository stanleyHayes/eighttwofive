import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { AccountPage } from "./AccountPage";
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

const me = { id: "u1", email: "ama@example.com", name: "Ama", role: "customer" as const };

function makeOrder(status: OrderStatus): Order {
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
    quote: { pricePesewas: 0, timeline: "", notes: "" },
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

function mockFetch(orders: Order[]) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";

    if (url.includes("/api/v1/auth/me") && method === "GET") {
      return jsonResponse(200, { data: { user: me } });
    }
    if (url.includes("/api/v1/settings") && method === "GET") {
      return jsonResponse(200, { data: defaultSettings });
    }
    if (url.includes("/api/v1/orders") && method === "GET" && !url.includes("/api/v1/orders/")) {
      return jsonResponse(200, { data: orders });
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
        <AccountPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AccountPage orders list", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders each order with ref, design name, stage and date", async () => {
    vi.stubGlobal("fetch", mockFetch([makeOrder("booked")]));
    renderPage();

    expect(await screen.findByText("Midnight Blazer")).toBeInTheDocument();
    expect(screen.getByText("E25-0001")).toBeInTheDocument();
    expect(screen.getByText("order confirmed")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /view order/i })).toHaveAttribute("href", "/account/orders/E25-0001");
  });

  it("shows an empty state when the customer has no orders", async () => {
    vi.stubGlobal("fetch", mockFetch([]));
    renderPage();

    expect(await screen.findByText(/no orders yet/i)).toBeInTheDocument();
  });
});

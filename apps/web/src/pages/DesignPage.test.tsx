import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router";
import { DesignPage } from "./DesignPage";

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

const design = {
  id: "d2",
  collectionId: "c1",
  name: "Osu Gown",
  slug: "osu-gown",
  note: "Bias-cut, floor length.",
  photos: [],
  sizeBands: [
    { label: "8", pricePesewas: 38000, chart: { bust: "86", waist: "70" } },
    { label: "10", pricePesewas: 42000, chart: { bust: "90", waist: "74" } },
  ],
  status: "live",
  createdAt: "2026-03-02T10:00:00Z",
};

function mockDesignFetch() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(String(input), "http://localhost");
    const method = init?.method ?? "GET";
    if (method === "GET" && url.pathname === "/api/v1/designs/osu-gown") {
      return jsonResponse(200, { data: design });
    }
    if (method === "GET" && url.pathname === "/api/v1/settings") {
      return jsonResponse(200, {
        data: {
          depositPesewas: 20000,
          whatsappNumber: "",
          visitLocation: "",
          cloudName: "",
          deliveryRates: [{ area: "East Legon", ratePesewas: 2000 }],
        },
      });
    }
    if (method === "GET" && url.pathname === "/api/v1/healthz") {
      return jsonResponse(200, { data: { status: "ok" } });
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
      <MemoryRouter initialEntries={["/designs/osu-gown"]}>
        <Routes>
          <Route path="/designs/:slug" element={<DesignPage />} />
          <Route path="/account/orders/:ref" element={<div data-testid="order-detail">Order detail</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function fillCustomerDetails() {
  fireEvent.change(screen.getByRole("textbox", { name: /full name/i }), { target: { value: "Ama" } });
  fireEvent.change(screen.getByRole("textbox", { name: /email/i }), { target: { value: "ama@example.com" } });
  fireEvent.change(screen.getByRole("textbox", { name: /phone number/i }), { target: { value: "+233200000000" } });
}

describe("DesignPage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders the size bands and switching band updates price and chart", async () => {
    vi.stubGlobal("fetch", mockDesignFetch());
    renderPage();

    expect(await screen.findByRole("heading", { name: "Osu Gown" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "8", pressed: true })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "10", pressed: false })).toBeInTheDocument();

    // First band selected by default.
    expect(screen.getByText("GH₵ 380.00")).toBeInTheDocument();
    expect(screen.getByText("86")).toBeInTheDocument();
    expect(screen.getByText("70")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "10" }));

    expect(await screen.findByText("GH₵ 420.00")).toBeInTheDocument();
    expect(screen.getByText("90")).toBeInTheDocument();
    expect(screen.getByText("74")).toBeInTheDocument();
    expect(screen.queryByText("GH₵ 380.00")).not.toBeInTheDocument();
    expect(screen.queryByText("86")).not.toBeInTheDocument();

    expect(screen.getByRole("button", { name: /order this design/i })).toBeInTheDocument();
  });

  it("creates a standard order and redirects to the payment URL", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(String(input), "http://localhost");
      const method = init?.method ?? "GET";
      if (method === "POST" && url.pathname === "/api/v1/orders") {
        return jsonResponse(201, {
          data: {
            order: { ref: "E25-STD", status: "pending_payment" },
            paymentUrl: "https://checkout.test/pay",
          },
        });
      }
      return mockDesignFetch()(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderPage();
    expect(await screen.findByRole("heading", { name: "Osu Gown" })).toBeInTheDocument();

    fillCustomerDetails();
    fireEvent.click(screen.getByRole("button", { name: /order this design/i }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining("/api/v1/orders"),
        expect.objectContaining({
          method: "POST",
          body: expect.stringContaining("\"bandLabel\":\"8\""),
        }),
      );
    });
  });

  it("hides price and routes to the custom request flow when a custom option is chosen", async () => {
    vi.stubGlobal("fetch", mockDesignFetch());
    renderPage();

    expect(await screen.findByRole("heading", { name: "Osu Gown" })).toBeInTheDocument();
    expect(screen.getByText("GH₵ 380.00")).toBeInTheDocument();

    await userEvent.click(screen.getByLabelText("My size isn't listed"));
    await userEvent.click(screen.getByLabelText("Measure yourself"));

    expect(screen.queryByText("GH₵ 380.00")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /send request/i })).toBeInTheDocument();
  });

  it("submits a self-measure custom request and shows an inline confirmation", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(String(input), "http://localhost");
      const method = init?.method ?? "GET";
      if (method === "POST" && url.pathname === "/api/v1/orders/request") {
        return jsonResponse(201, {
          data: {
            order: { ref: "E25-CUSTOM", status: "requested", type: "custom_size" },
          },
        });
      }
      return mockDesignFetch()(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderPage();
    expect(await screen.findByRole("heading", { name: "Osu Gown" })).toBeInTheDocument();

    await userEvent.click(screen.getByLabelText("My size isn't listed"));
    await userEvent.click(screen.getByLabelText("Measure yourself"));

    fireEvent.change(screen.getByRole("textbox", { name: /bust/i }), { target: { value: "90 cm" } });
    fireEvent.change(screen.getByRole("textbox", { name: /waist/i }), { target: { value: "74 cm" } });
    fireEvent.change(screen.getByRole("textbox", { name: /hips/i }), { target: { value: "96 cm" } });
    fireEvent.change(screen.getByRole("textbox", { name: /length/i }), { target: { value: "120 cm" } });

    fillCustomerDetails();
    fireEvent.click(screen.getByRole("button", { name: /send request/i }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining("/api/v1/orders/request"),
        expect.objectContaining({
          method: "POST",
          body: expect.stringContaining("\"sizeMode\":\"self\""),
        }),
      );
    });

    // Checkout is anonymous (no session), so we confirm inline instead of
    // navigating to the auth-gated order page.
    expect(await screen.findByText(/request received/i)).toBeInTheDocument();
  });

  it("copies the page link and shows a success state", async () => {
    vi.stubGlobal("fetch", mockDesignFetch());
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(window.navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });
    renderPage();

    expect(await screen.findByRole("heading", { name: "Osu Gown" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /copy link/i }));

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith(window.location.href);
    });
    expect(await screen.findByRole("button", { name: /link copied/i })).toBeInTheDocument();
  });

  it("shows an error state when the clipboard write fails", async () => {
    vi.stubGlobal("fetch", mockDesignFetch());
    const writeText = vi.fn().mockRejectedValue(new Error("denied"));
    Object.defineProperty(window.navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });
    renderPage();

    expect(await screen.findByRole("heading", { name: "Osu Gown" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /copy link/i }));

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith(window.location.href);
    });
    expect(await screen.findByRole("button", { name: /could not copy link/i })).toBeInTheDocument();
  });
});

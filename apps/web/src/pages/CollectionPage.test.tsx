import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router";
import { CollectionPage } from "./CollectionPage";

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

function mock404Fetch() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(String(input), "http://localhost");
    const method = init?.method ?? "GET";
    if (method === "GET" && url.pathname === "/api/v1/collections/faded-archive") {
      return jsonResponse(404, {
        error: { code: "not_found", message: "collection not found" },
      });
    }
    if (method === "GET" && url.pathname === "/api/v1/settings") {
      return jsonResponse(200, {
        data: { depositPesewas: 0, whatsappNumber: "", visitLocation: "", cloudName: "" },
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
      <MemoryRouter initialEntries={["/collections/faded-archive"]}>
        <Routes>
          <Route path="/collections/:slug" element={<CollectionPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("CollectionPage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders the friendly retired panel on a 404 and links back to the store", async () => {
    vi.stubGlobal("fetch", mock404Fetch());
    renderPage();

    expect(
      await screen.findByRole("heading", { name: /this collection has been retired/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/limited run/i)).toBeInTheDocument();

    const backLink = screen.getByRole("link", { name: /back to the store/i });
    expect(backLink).toHaveAttribute("href", "/store");
  });
});

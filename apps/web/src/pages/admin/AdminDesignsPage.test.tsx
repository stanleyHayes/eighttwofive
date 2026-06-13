import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { seedAdmin } from "@/test/seedAdmin";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { AdminDesignsPage } from "./AdminDesignsPage";

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

const designs = [
  {
    id: "d1",
    collectionId: "c1",
    name: "Sika Dress",
    slug: "sika-dress",
    note: "",
    photos: [],
    sizeBands: [{ label: "8", pricePesewas: 50000, chart: {} }],
    status: "live",
    createdAt: "2026-03-01T10:00:00Z",
  },
  {
    id: "d2",
    collectionId: "c1",
    name: "Osu Gown",
    slug: "osu-gown",
    note: "",
    photos: [],
    sizeBands: [
      { label: "8", pricePesewas: 38000, chart: {} },
      { label: "10", pricePesewas: 42000, chart: {} },
    ],
    status: "live",
    createdAt: "2026-03-02T10:00:00Z",
  },
];

function mockCatalogFetch() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";
    if (url.includes("/api/v1/admin/uploads/sign")) {
      return jsonResponse(503, {
        error: { code: "not_configured", message: "uploads not configured" },
      });
    }
    if (url.includes("/api/v1/admin/designs/retire") && method === "POST") {
      return jsonResponse(200, { data: { status: "ok" } });
    }
    if (url.includes("/api/v1/admin/designs/restore") && method === "POST") {
      return jsonResponse(200, { data: { status: "ok" } });
    }
    if (url.includes("/api/v1/admin/designs") && method === "GET") {
      return jsonResponse(200, {
        data: { items: designs, total: designs.length, page: 1, pageSize: 20 },
      });
    }
    if (url.includes("/api/v1/admin/collections") && method === "GET") {
      return jsonResponse(200, {
        data: {
          items: [
            {
              id: "c1",
              name: "Accra Nights",
              slug: "accra-nights",
              note: "",
              status: "live",
              createdAt: "2026-01-05T10:00:00Z",
            },
          ],
          total: 1,
          page: 1,
          pageSize: 100,
        },
      });
    }
    return jsonResponse(404, { error: { code: "not_found", message: "no route" } });
  });
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  seedAdmin(client);
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/admin/designs"]}>
        <AdminDesignsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AdminDesignsPage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("bulk retires the selected designs with {ids}", async () => {
    const fetchSpy = mockCatalogFetch();
    vi.stubGlobal("fetch", fetchSpy);
    renderPage();

    expect(await screen.findByText("Sika Dress")).toBeInTheDocument();
    expect(screen.getByText("Osu Gown")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("checkbox", { name: /select sika dress/i }));
    await userEvent.click(screen.getByRole("checkbox", { name: /select osu gown/i }));
    await userEvent.click(screen.getByRole("button", { name: /retire selected/i }));

    await waitFor(() => {
      const retireCall = fetchSpy.mock.calls.find(
        ([url, init]) =>
          String(url).includes("/api/v1/admin/designs/retire") && init?.method === "POST",
      );
      expect(retireCall).toBeTruthy();
    });

    const retireCall = fetchSpy.mock.calls.find(
      ([url, init]) =>
        String(url).includes("/api/v1/admin/designs/retire") && init?.method === "POST",
    );
    expect(JSON.parse(retireCall![1]!.body as string)).toEqual({ ids: ["d1", "d2"] });
  });
});

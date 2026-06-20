import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, useLocation } from "react-router";
import { StorePage } from "./StorePage";

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

const collections = [
  {
    id: "c1",
    name: "Accra Nights",
    slug: "accra-nights",
    note: "Ten designs for long evenings.",
    status: "live",
    createdAt: "2026-01-05T10:00:00Z",
  },
  {
    id: "c2",
    name: "Harmattan",
    slug: "harmattan",
    note: "",
    status: "live",
    createdAt: "2026-02-05T10:00:00Z",
  },
];

const designs = [
  {
    id: "d1",
    collectionId: "c1",
    name: "Sika Dress",
    slug: "sika-dress",
    note: "",
    photos: [{ publicId: "store/sika", order: 0 }],
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

function mockStoreFetch() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(String(input), "http://localhost");
    const method = init?.method ?? "GET";
    if (method !== "GET") {
      return jsonResponse(404, { error: { code: "not_found", message: "no route" } });
    }
    if (url.pathname === "/api/v1/settings") {
      return jsonResponse(200, {
        data: { depositPesewas: 20000, whatsappNumber: "", visitLocation: "", cloudName: "demo" },
      });
    }
    if (url.pathname === "/api/v1/collections") {
      return jsonResponse(200, { data: collections });
    }
    if (url.pathname === "/api/v1/designs") {
      const q = (url.searchParams.get("q") ?? "").toLowerCase();
      return jsonResponse(200, {
        data: designs.filter((design) => design.name.toLowerCase().includes(q)),
      });
    }
    if (url.pathname === "/api/v1/healthz") {
      return jsonResponse(200, { data: { status: "ok" } });
    }
    return jsonResponse(404, { error: { code: "not_found", message: "no route" } });
  });
}

const manyDesigns = Array.from({ length: 15 }, (_, i) => ({
  id: `m${i + 1}`,
  collectionId: "c1",
  name: `Design ${String(i + 1).padStart(2, "0")}`,
  slug: `design-${i + 1}`,
  note: "",
  photos: [],
  sizeBands: [{ label: "8", pricePesewas: 50000, chart: {} }],
  status: "live",
  createdAt: "2026-03-01T10:00:00Z",
}));

function mockManyDesignsFetch() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(String(input), "http://localhost");
    if ((init?.method ?? "GET") !== "GET") {
      return jsonResponse(404, { error: { code: "not_found", message: "no route" } });
    }
    if (url.pathname === "/api/v1/settings") {
      return jsonResponse(200, {
        data: { depositPesewas: 20000, whatsappNumber: "", visitLocation: "", cloudName: "demo" },
      });
    }
    if (url.pathname === "/api/v1/collections") {
      return jsonResponse(200, { data: [] });
    }
    if (url.pathname === "/api/v1/designs") {
      return jsonResponse(200, { data: manyDesigns });
    }
    return jsonResponse(404, { error: { code: "not_found", message: "no route" } });
  });
}

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location-search">{location.search}</div>;
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/store"]}>
        <StorePage />
        <LocationProbe />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("StorePage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders live collections and designs with from-prices", async () => {
    vi.stubGlobal("fetch", mockStoreFetch());
    renderPage();

    expect(await screen.findByRole("heading", { name: "Accra Nights" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Harmattan" })).toBeInTheDocument();
    expect(screen.getByText("Ten designs for long evenings.")).toBeInTheDocument();

    expect(await screen.findByRole("heading", { name: "Sika Dress" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Osu Gown" })).toBeInTheDocument();
    // Single band shows the price; multiple bands show the minimum as a from-price.
    expect(screen.getByText("GH₵ 500.00")).toBeInTheDocument();
    expect(screen.getByText("From GH₵ 380.00")).toBeInTheDocument();
  });

  it("debounces search into ?q= and refetches the design grid", async () => {
    const fetchSpy = mockStoreFetch();
    vi.stubGlobal("fetch", fetchSpy);
    renderPage();

    expect(await screen.findByRole("heading", { name: "Sika Dress" })).toBeInTheDocument();

    await userEvent.type(screen.getByRole("textbox", { name: /search designs/i }), "gown");

    // After the 300ms debounce the page URL carries the term…
    await waitFor(
      () => {
        expect(screen.getByTestId("location-search")).toHaveTextContent("?q=gown");
      },
      { timeout: 3000 },
    );
    // …and the public designs endpoint is queried with it.
    await waitFor(
      () => {
        const searchCall = fetchSpy.mock.calls.find(([url]) =>
          String(url).includes("/api/v1/designs?q=gown"),
        );
        expect(searchCall).toBeTruthy();
      },
      { timeout: 3000 },
    );

    expect(await screen.findByRole("heading", { name: "Osu Gown" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.queryByRole("heading", { name: "Sika Dress" })).not.toBeInTheDocument();
    });
  });

  it("reveals more designs with Load more", async () => {
    vi.stubGlobal("fetch", mockManyDesignsFetch());
    renderPage();

    // The first page (12) is shown; the 13th is withheld behind Load more.
    expect(await screen.findByRole("heading", { name: "Design 01" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Design 12" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Design 13" })).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /load more/i }));

    // The rest reveal and the button retires once everything is shown.
    expect(await screen.findByRole("heading", { name: "Design 13" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Design 15" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /load more/i })).not.toBeInTheDocument();
  });
});

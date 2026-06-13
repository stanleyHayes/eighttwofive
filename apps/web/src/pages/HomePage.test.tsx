import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { HomePage } from "./HomePage";

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

const collections = [
  {
    id: "c0",
    name: "Archive",
    slug: "archive",
    note: "Older designs.",
    status: "live",
    createdAt: "2025-12-01T10:00:00Z",
  },
  {
    id: "c1",
    name: "First Drop",
    slug: "first-drop",
    note: "The opening collection.",
    status: "live",
    createdAt: "2026-01-05T10:00:00Z",
  },
  {
    id: "c2",
    name: "Accra Nights",
    slug: "accra-nights",
    note: "Evening wear.",
    status: "live",
    createdAt: "2026-03-05T10:00:00Z",
  },
  {
    id: "c3",
    name: "Harmattan",
    slug: "harmattan",
    note: "Dry-season tones.",
    status: "live",
    createdAt: "2026-02-05T10:00:00Z",
  },
];

const designs = [
  {
    id: "d1",
    collectionId: "c1",
    name: "Early Dress",
    slug: "early-dress",
    note: "",
    photos: [{ publicId: "store/early", order: 0 }],
    sizeBands: [{ label: "8", pricePesewas: 40000, chart: {} }],
    status: "live",
    createdAt: "2026-01-10T10:00:00Z",
  },
  {
    id: "d2",
    collectionId: "c2",
    name: "Sika Dress",
    slug: "sika-dress",
    note: "",
    photos: [{ publicId: "store/sika", order: 0 }],
    sizeBands: [{ label: "8", pricePesewas: 50000, chart: {} }],
    status: "live",
    createdAt: "2026-03-10T10:00:00Z",
  },
  {
    id: "d3",
    collectionId: "c2",
    name: "Osu Gown",
    slug: "osu-gown",
    note: "",
    photos: [],
    sizeBands: [
      { label: "8", pricePesewas: 38000, chart: {} },
      { label: "10", pricePesewas: 42000, chart: {} },
    ],
    status: "live",
    createdAt: "2026-03-11T10:00:00Z",
  },
  {
    id: "d4",
    collectionId: "c3",
    name: "Harmattan Blazer",
    slug: "harmattan-blazer",
    note: "",
    photos: [],
    sizeBands: [{ label: "10", pricePesewas: 45000, chart: {} }],
    status: "live",
    createdAt: "2026-02-12T10:00:00Z",
  },
  {
    id: "d5",
    collectionId: "c3",
    name: "Dry Season Skirt",
    slug: "dry-season-skirt",
    note: "",
    photos: [],
    sizeBands: [{ label: "8", pricePesewas: 35000, chart: {} }],
    status: "live",
    createdAt: "2026-02-13T10:00:00Z",
  },
];

function mockHomeFetch() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(String(input), "http://localhost");
    const method = init?.method ?? "GET";

    if (method === "POST" && url.pathname === "/api/v1/waitlist") {
      return jsonResponse(201, {
        data: { id: "s1", name: "Ada", email: "ada@example.com", createdAt: "2026-01-01T00:00:00Z" },
      });
    }

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
      return jsonResponse(200, { data: designs });
    }
    if (url.pathname === "/api/v1/healthz") {
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
      <MemoryRouter initialEntries={["/"]}>
        <HomePage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("HomePage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders featured collections and newest designs, and links to the store", async () => {
    vi.stubGlobal("fetch", mockHomeFetch());
    renderPage();

    // Hero + CTA.
    expect(await screen.findByRole("heading", { name: /made-to-measure womenswear/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /shop the store/i })).toHaveAttribute("href", "/store");

    // Only the 3 newest collections are featured (sorted desc: Accra Nights, Harmattan, First Drop).
    expect(await screen.findByRole("heading", { name: "Accra Nights" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Harmattan" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "First Drop" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Archive" })).not.toBeInTheDocument();

    // Only the 4 newest designs are featured.
    expect(await screen.findByRole("heading", { name: "Sika Dress" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Osu Gown" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Harmattan Blazer" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Dry Season Skirt" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Early Dress" })).not.toBeInTheDocument();

    // Store CTA.
    expect(screen.getByRole("link", { name: /shop now/i })).toHaveAttribute("href", "/store");
  });

  it("submits the waitlist form from the bottom section", async () => {
    const fetchSpy = mockHomeFetch();
    vi.stubGlobal("fetch", fetchSpy);
    renderPage();

    expect(await screen.findByRole("heading", { name: /be the first to know/i })).toBeInTheDocument();

    await userEvent.type(screen.getByLabelText(/name/i), "Ada");
    await userEvent.type(screen.getByLabelText(/email/i), "ada@example.com");
    await userEvent.click(screen.getByRole("button", { name: /join the waitlist/i }));

    await waitFor(() => {
      expect(screen.getByRole("status")).toHaveTextContent(/ada@example.com/i);
    });

    const waitlistCall = fetchSpy.mock.calls.find(([url, init]) => {
      const u = new URL(String(url), "http://localhost");
      return u.pathname === "/api/v1/waitlist" && init?.method === "POST";
    });
    expect(waitlistCall).toBeTruthy();
  });
});

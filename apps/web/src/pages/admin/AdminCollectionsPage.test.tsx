import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { AdminCollectionsPage } from "./AdminCollectionsPage";

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
    note: "evening wear",
    status: "live",
    createdAt: "2026-01-05T10:00:00Z",
  },
  {
    id: "c2",
    name: "Harmattan Capsule",
    slug: "harmattan-capsule",
    note: "",
    status: "retired",
    createdAt: "2025-11-20T10:00:00Z",
    retiredAt: "2026-02-01T10:00:00Z",
  },
];

function mockCatalogFetch() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";
    if (url.includes("/api/v1/admin/collections") && method === "GET") {
      return jsonResponse(200, {
        data: { items: collections, total: collections.length, page: 1, pageSize: 20 },
      });
    }
    if (url.includes("/api/v1/admin/collections") && method === "POST") {
      return jsonResponse(201, {
        data: {
          id: "c9",
          name: "Volta",
          slug: "volta",
          note: "Riverside edit",
          status: "live",
          createdAt: "2026-06-01T10:00:00Z",
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
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/admin/collections"]}>
        <AdminCollectionsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AdminCollectionsPage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders collection rows from the API", async () => {
    vi.stubGlobal("fetch", mockCatalogFetch());
    renderPage();

    expect(await screen.findByText("Accra Nights")).toBeInTheDocument();
    expect(screen.getByText("Harmattan Capsule")).toBeInTheDocument();
    expect(screen.getByText("accra-nights")).toBeInTheDocument();
    expect(screen.getByText("harmattan-capsule")).toBeInTheDocument();
    expect(screen.getByText("live")).toBeInTheDocument();
    expect(screen.getByText("retired")).toBeInTheDocument();
  });

  it("creates a collection and posts name and note", async () => {
    const fetchSpy = mockCatalogFetch();
    vi.stubGlobal("fetch", fetchSpy);
    renderPage();

    await screen.findByText("Accra Nights");
    await userEvent.click(screen.getByRole("button", { name: /new collection/i }));

    await userEvent.type(screen.getByLabelText(/name/i), "Volta");
    await userEvent.type(screen.getByLabelText(/note/i), "Riverside edit");
    await userEvent.click(screen.getByRole("button", { name: /create collection/i }));

    await waitFor(() => {
      const postCall = fetchSpy.mock.calls.find(([, init]) => init?.method === "POST");
      expect(postCall).toBeTruthy();
    });

    const postCall = fetchSpy.mock.calls.find(([, init]) => init?.method === "POST");
    expect(String(postCall![0])).toContain("/api/v1/admin/collections");
    expect(JSON.parse(postCall![1]!.body as string)).toEqual({
      name: "Volta",
      note: "Riverside edit",
    });
  });
});

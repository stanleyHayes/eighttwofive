import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { AdminSubscribersPage } from "./AdminSubscribersPage";

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

const subscribers = [
  { id: "s1", email: "ada@example.com", name: "Ada Lovelace", createdAt: "2026-05-01T10:00:00Z" },
  { id: "s2", email: "grace@example.com", name: "Grace Hopper", createdAt: "2026-04-20T10:00:00Z" },
];

function mockFetch() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";

    if (url.includes("/api/v1/admin/waitlist") && method === "GET") {
      return jsonResponse(200, {
        data: { items: subscribers, total: subscribers.length, page: 1, pageSize: 20 },
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
      <MemoryRouter initialEntries={["/admin/subscribers"]}>
        <AdminSubscribersPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AdminSubscribersPage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders subscriber rows from the paginated response", async () => {
    vi.stubGlobal("fetch", mockFetch());
    renderPage();

    expect(await screen.findByText("ada@example.com")).toBeInTheDocument();
    expect(screen.getByText("Ada Lovelace")).toBeInTheDocument();
    expect(screen.getByText("grace@example.com")).toBeInTheDocument();
    expect(screen.getByText("Grace Hopper")).toBeInTheDocument();
    expect(screen.getByText(/2 subscribers/i)).toBeInTheDocument();
  });

  it("renders an enabled Export CSV button once rows load", async () => {
    vi.stubGlobal("fetch", mockFetch());
    renderPage();

    await screen.findByText("ada@example.com");

    const exportButton = screen.getByRole("button", { name: /export csv/i });
    expect(exportButton).toBeInTheDocument();
    expect(exportButton).toBeEnabled();
  });
});

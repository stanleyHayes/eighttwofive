import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router";
import { AdminDesignEditorPage } from "./AdminDesignEditorPage";

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

function mockCatalogFetch() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";
    if (url.includes("/api/v1/admin/uploads/sign")) {
      return jsonResponse(503, {
        error: { code: "not_configured", message: "uploads not configured" },
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

function designSaveCalls(fetchSpy: ReturnType<typeof mockCatalogFetch>) {
  return fetchSpy.mock.calls.filter(([url, init]) => {
    const u = String(url);
    const method = init?.method ?? "GET";
    return u.includes("/api/v1/admin/designs") && (method === "POST" || method === "PUT");
  });
}

function renderEditor() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/admin/designs/new"]}>
        <Routes>
          <Route path="/admin/designs/new" element={<AdminDesignEditorPage />} />
          <Route path="/admin/designs" element={<div>designs list</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AdminDesignEditorPage validation", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("rejects an empty name and missing collection without saving", async () => {
    const fetchSpy = mockCatalogFetch();
    vi.stubGlobal("fetch", fetchSpy);
    renderEditor();

    await userEvent.click(await screen.findByRole("button", { name: /save design/i }));

    expect(await screen.findByText("Name is required.")).toBeInTheDocument();
    expect(screen.getByText("Choose a collection.")).toBeInTheDocument();
    expect(designSaveCalls(fetchSpy)).toHaveLength(0);
  });

  it("requires at least one size band without saving", async () => {
    const fetchSpy = mockCatalogFetch();
    vi.stubGlobal("fetch", fetchSpy);
    renderEditor();

    await screen.findByRole("button", { name: /save design/i });
    await userEvent.click(screen.getByRole("button", { name: /remove size band 1/i }));
    await userEvent.click(screen.getByRole("button", { name: /save design/i }));

    expect(await screen.findByText("Add at least one size band.")).toBeInTheDocument();
    expect(designSaveCalls(fetchSpy)).toHaveLength(0);
  });

  it("rejects duplicate band labels without saving", async () => {
    const fetchSpy = mockCatalogFetch();
    vi.stubGlobal("fetch", fetchSpy);
    renderEditor();

    await screen.findByRole("button", { name: /save design/i });
    await userEvent.click(screen.getByRole("button", { name: /add size band/i }));

    const labelFields = screen.getAllByLabelText(/^label/i);
    expect(labelFields).toHaveLength(2);
    await userEvent.type(labelFields[0], "8");
    await userEvent.type(labelFields[1], "8");
    await userEvent.click(screen.getByRole("button", { name: /save design/i }));

    const duplicateErrors = await screen.findAllByText("Band labels must be unique.");
    expect(duplicateErrors.length).toBeGreaterThan(0);
    expect(designSaveCalls(fetchSpy)).toHaveLength(0);
  });
});

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
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

const adminUser = {
  id: "u1",
  email: "boss@e25.com",
  name: "Boss",
  role: "admin",
  permissions: ["subscribers:read", "subscribers:write"],
  isSuperAdmin: true,
};

// canWrite=false by default (no /auth/me); pass an admin to enable the delete UI.
function mockFetch(user?: typeof adminUser) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";

    if (url.includes("/auth/me")) {
      return user
        ? jsonResponse(200, { data: { user } })
        : jsonResponse(401, { error: { code: "unauthorized", message: "no" } });
    }
    if (url.includes("/admin/waitlist/") && method === "DELETE") return jsonResponse(204, {});
    if (url.includes("/admin/waitlist") && method === "GET") {
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

  it("renders subscriber rows and the summary stats", async () => {
    vi.stubGlobal("fetch", mockFetch());
    renderPage();

    expect(await screen.findByText("ada@example.com")).toBeInTheDocument();
    expect(screen.getByText("Ada Lovelace")).toBeInTheDocument();
    expect(screen.getByText("grace@example.com")).toBeInTheDocument();
    expect(screen.getByText("Newest signup")).toBeInTheDocument();
  });

  it("renders an enabled Export CSV button once rows load", async () => {
    vi.stubGlobal("fetch", mockFetch());
    renderPage();

    await screen.findByText("ada@example.com");

    const exportButton = screen.getByRole("button", { name: /export csv/i });
    expect(exportButton).toBeInTheDocument();
    expect(exportButton).toBeEnabled();
  });

  it("deletes a subscriber after confirming (with subscribers:write)", async () => {
    const fetchMock = mockFetch(adminUser);
    vi.stubGlobal("fetch", fetchMock);
    renderPage();

    await screen.findByText("ada@example.com");

    fireEvent.click(await screen.findByRole("button", { name: "Remove ada@example.com" }));
    fireEvent.click(screen.getByRole("button", { name: "Remove" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining("/admin/waitlist/s1"),
        expect.objectContaining({ method: "DELETE" }),
      );
    });
  });

  it("hides delete actions without subscribers:write", async () => {
    vi.stubGlobal("fetch", mockFetch());
    renderPage();

    await screen.findByText("ada@example.com");
    expect(screen.queryByRole("button", { name: "Remove ada@example.com" })).not.toBeInTheDocument();
  });
});

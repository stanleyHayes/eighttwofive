import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { AdminTeamPage } from "./AdminTeamPage";

function jsonResponse(status: number, body: unknown) {
  return { ok: status >= 200 && status < 300, status, json: () => Promise.resolve(body) };
}

const admin = {
  id: "u1",
  email: "boss@e25.com",
  name: "Boss",
  role: "admin",
  permissions: ["team:read", "team:write"],
  isSuperAdmin: true,
};

const partner = {
  id: "u2",
  email: "partner@e25.com",
  name: "Partner",
  role: "staff",
  permissions: [],
  isSuperAdmin: false,
};

const roles = [
  { key: "customer", name: "Customer", description: "", permissions: [], system: true, adminArea: false },
  { key: "staff", name: "Staff", description: "", permissions: [], system: true, adminArea: true },
  { key: "photographer", name: "Photographer", description: "", permissions: [], system: false, adminArea: true },
];

function mockFetch() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";
    if (url.includes("/auth/me")) return jsonResponse(200, { data: { user: admin } });
    if (url.includes("/admin/users") && method === "GET") {
      return jsonResponse(200, { data: { items: [partner], total: 1, page: 1, pageSize: 20 } });
    }
    if (url.includes("/admin/roles") && method === "GET") return jsonResponse(200, { data: roles });
    return jsonResponse(404, { error: { code: "not_found", message: "no route" } });
  });
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/admin/team"]}>
        <AdminTeamPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AdminTeamPage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("populates the role dropdown from the roles API and shows the invite action", async () => {
    vi.stubGlobal("fetch", mockFetch());
    renderPage();

    expect(await screen.findByText("partner@e25.com")).toBeInTheDocument();

    // The dropdown options come from the store, including the custom role and a
    // storefront-only marker for non-dashboard roles.
    expect(await screen.findByRole("option", { name: "Photographer" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: /Customer — storefront only/ })).toBeInTheDocument();

    expect(screen.getByRole("button", { name: "Invite partner" })).toBeInTheDocument();
  });
});

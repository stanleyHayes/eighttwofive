import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { AdminRolesPage } from "./AdminRolesPage";

function jsonResponse(status: number, body: unknown) {
  return { ok: status >= 200 && status < 300, status, json: () => Promise.resolve(body) };
}

const adminUser = {
  id: "u1",
  email: "boss@e25.com",
  name: "Boss",
  role: "admin",
  permissions: ["team:read", "team:write"],
  isSuperAdmin: true,
};

const roles = [
  {
    key: "admin",
    name: "Admin",
    description: "Full access",
    permissions: ["team:read", "team:write", "orders:read"],
    system: true,
    adminArea: true,
  },
  {
    key: "photographer",
    name: "Photographer",
    description: "Shoots the lookbook",
    permissions: ["catalogue:read"],
    system: false,
    adminArea: true,
  },
];

const permissions = [
  { key: "orders:read", label: "View orders", description: "See orders", group: "Orders" },
  { key: "team:write", label: "Manage team", description: "Invite & assign", group: "Team" },
];

function mockFetch(user: typeof adminUser = adminUser) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";
    if (url.includes("/auth/me")) return jsonResponse(200, { data: { user } });
    if (url.includes("/admin/roles") && method === "GET") return jsonResponse(200, { data: roles });
    if (url.includes("/admin/permissions")) return jsonResponse(200, { data: permissions });
    return jsonResponse(404, { error: { code: "not_found", message: "no route" } });
  });
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/admin/roles"]}>
        <AdminRolesPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AdminRolesPage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("lists roles with a built-in badge and a new-role action for admins", async () => {
    vi.stubGlobal("fetch", mockFetch());
    renderPage();

    expect(await screen.findByRole("heading", { name: "Photographer" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Admin" })).toBeInTheDocument();
    expect(screen.getByText("Built-in")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New role" })).toBeInTheDocument();
  });

  it("hides the new-role action for a user without team:write", async () => {
    const viewer = { ...adminUser, role: "viewer", permissions: ["team:read"], isSuperAdmin: false };
    vi.stubGlobal("fetch", mockFetch(viewer));
    renderPage();

    expect(await screen.findByText("Photographer")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "New role" })).not.toBeInTheDocument();
  });
});

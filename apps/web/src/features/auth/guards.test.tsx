import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router";
import type { ReactElement } from "react";
import type { User } from "@/lib/api";
import { AuthGuard, AdminGuard } from "./guards";

function mockFetch(status: number, body: unknown) {
  return vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  });
}

function renderGuarded(initialPath: string, guarded: ReactElement) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route path="/" element={<div>landing page</div>} />
          <Route path="/login" element={<div>login page</div>} />
          <Route path={initialPath} element={guarded} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const customer: User = {
  id: "u1",
  email: "ama@example.com",
  name: "Ama",
  role: "customer",
};

describe("AuthGuard", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("redirects to /login when /auth/me returns 401", async () => {
    vi.stubGlobal(
      "fetch",
      mockFetch(401, { error: { code: "unauthorized", message: "no session" } }),
    );
    renderGuarded(
      "/account",
      <AuthGuard>
        <div>secret account</div>
      </AuthGuard>,
    );

    expect(await screen.findByText("login page")).toBeInTheDocument();
    expect(screen.queryByText("secret account")).not.toBeInTheDocument();
  });

  it("renders children when a session exists", async () => {
    vi.stubGlobal("fetch", mockFetch(200, { data: { user: customer } }));
    renderGuarded(
      "/account",
      <AuthGuard>
        <div>secret account</div>
      </AuthGuard>,
    );

    expect(await screen.findByText("secret account")).toBeInTheDocument();
  });
});

describe("AdminGuard", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("redirects non-admin users to the landing page", async () => {
    vi.stubGlobal("fetch", mockFetch(200, { data: { user: customer } }));
    renderGuarded(
      "/admin",
      <AdminGuard>
        <div>admin shell</div>
      </AdminGuard>,
    );

    expect(await screen.findByText("landing page")).toBeInTheDocument();
    expect(screen.queryByText("admin shell")).not.toBeInTheDocument();
  });

  it("renders children for admins", async () => {
    vi.stubGlobal(
      "fetch",
      mockFetch(200, { data: { user: { ...customer, role: "admin" } } }),
    );
    renderGuarded(
      "/admin",
      <AdminGuard>
        <div>admin shell</div>
      </AdminGuard>,
    );

    expect(await screen.findByText("admin shell")).toBeInTheDocument();
  });
});

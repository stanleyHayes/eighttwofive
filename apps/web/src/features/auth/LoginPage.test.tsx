import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import type { ReactElement } from "react";
import { LoginPage } from "./LoginPage";

function renderWithProviders(ui: ReactElement) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/login"]}>{ui}</MemoryRouter>
    </QueryClientProvider>,
  );
}

function mockFetch(status: number, body: unknown) {
  return vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  });
}

describe("LoginPage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("validates name and email before submitting", async () => {
    const fetchSpy = mockFetch(202, { data: { status: "sent" } });
    vi.stubGlobal("fetch", fetchSpy);
    renderWithProviders(<LoginPage />);

    await userEvent.click(screen.getByRole("button", { name: /email me a link/i }));

    expect(await screen.findByText(/enter your name/i)).toBeInTheDocument();
    expect(fetchSpy).not.toHaveBeenCalled();

    await userEvent.type(screen.getByLabelText(/name/i), "Ama");
    await userEvent.type(screen.getByLabelText(/email/i), "not-an-email");
    await userEvent.click(screen.getByRole("button", { name: /email me a link/i }));

    expect(await screen.findByText(/valid email address/i)).toBeInTheDocument();
    expect(screen.queryByText(/enter your name/i)).not.toBeInTheDocument();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("submits name and email and shows the check-your-email panel on success", async () => {
    const fetchSpy = mockFetch(202, { data: { status: "sent" } });
    vi.stubGlobal("fetch", fetchSpy);
    renderWithProviders(<LoginPage />);

    await userEvent.type(screen.getByLabelText(/name/i), "Ama Hayford");
    await userEvent.type(screen.getByLabelText(/email/i), "Ama@Example.com");
    await userEvent.click(screen.getByRole("button", { name: /email me a link/i }));

    const status = await screen.findByRole("status");
    expect(status).toHaveTextContent(/check your email/i);
    expect(status).toHaveTextContent(/sign-in link/i);
    expect(status).toHaveTextContent(/ama@example.com/i);
    expect(fetchSpy).toHaveBeenCalledWith(
      "/api/v1/auth/request-link",
      expect.objectContaining({ method: "POST", credentials: "include" }),
    );

    const body = JSON.parse(fetchSpy.mock.calls[0][1].body as string);
    expect(body).toEqual({ email: "ama@example.com", name: "Ama Hayford" });
  });

  it("shows a server error when the request fails", async () => {
    vi.stubGlobal(
      "fetch",
      mockFetch(500, { error: { code: "internal", message: "boom" } }),
    );
    renderWithProviders(<LoginPage />);

    await userEvent.type(screen.getByLabelText(/name/i), "Ama");
    await userEvent.type(screen.getByLabelText(/email/i), "ama@example.com");
    await userEvent.click(screen.getByRole("button", { name: /email me a link/i }));

    expect(await screen.findByText(/something went wrong/i)).toBeInTheDocument();
  });
});

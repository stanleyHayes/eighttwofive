import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement } from "react";
import { WaitlistForm } from "./WaitlistForm";

function renderWithClient(ui: ReactElement) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

function mockFetch(status: number, body: unknown) {
  return vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  });
}

describe("WaitlistForm", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("validates email before submitting", async () => {
    const fetchSpy = mockFetch(201, {});
    vi.stubGlobal("fetch", fetchSpy);
    renderWithClient(<WaitlistForm />);

    await userEvent.type(screen.getByLabelText(/name/i), "Ada");
    await userEvent.type(screen.getByLabelText(/email/i), "not-an-email");
    await userEvent.click(screen.getByRole("button", { name: /join/i }));

    expect(await screen.findByText(/valid email address/i)).toBeInTheDocument();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("submits and shows confirmation on success", async () => {
    const subscriber = {
      id: "1",
      email: "ada@example.com",
      name: "Ada",
      createdAt: "2026-01-01T00:00:00Z",
    };
    vi.stubGlobal("fetch", mockFetch(201, { data: subscriber }));
    renderWithClient(<WaitlistForm />);

    await userEvent.type(screen.getByLabelText(/name/i), "Ada");
    await userEvent.type(screen.getByLabelText(/email/i), "Ada@Example.com");
    await userEvent.click(screen.getByRole("button", { name: /join/i }));

    expect(await screen.findByRole("status")).toHaveTextContent(/ada@example.com/i);
  });

  it("shows a friendly message when already subscribed", async () => {
    vi.stubGlobal(
      "fetch",
      mockFetch(409, { error: { code: "conflict", message: "duplicate" } }),
    );
    renderWithClient(<WaitlistForm />);

    await userEvent.type(screen.getByLabelText(/name/i), "Ada");
    await userEvent.type(screen.getByLabelText(/email/i), "ada@example.com");
    await userEvent.click(screen.getByRole("button", { name: /join/i }));

    expect(await screen.findByText(/already on the list/i)).toBeInTheDocument();
  });
});

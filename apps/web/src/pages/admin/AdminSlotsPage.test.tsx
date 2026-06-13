import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { AdminSlotsPage } from "./AdminSlotsPage";

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

const slots = [
  {
    id: "slot-1",
    start: "2026-07-01T10:00:00.000Z",
    end: "2026-07-01T11:00:00.000Z",
    status: "open",
    createdAt: "2026-06-01T00:00:00.000Z",
    updatedAt: "2026-06-01T00:00:00.000Z",
  },
];

const visits = [
  {
    id: "visit-1",
    orderId: "E25-VISIT-abc",
    slotId: "slot-1",
    depositPaymentId: "ps-1",
    status: "booked",
    createdAt: "2026-06-01T00:00:00.000Z",
    updatedAt: "2026-06-01T00:00:00.000Z",
  },
];

function mockSlotsFetch() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";

    if (url.includes("/api/v1/admin/slots") && method === "GET") {
      return jsonResponse(200, { data: slots });
    }
    if (url.includes("/api/v1/admin/visits") && method === "GET") {
      return jsonResponse(200, { data: visits });
    }
    if (url.includes("/api/v1/admin/slots") && method === "POST") {
      return jsonResponse(201, { data: { id: "slot-2", ...JSON.parse(init?.body as string) } });
    }
    if (url.includes("/api/v1/admin/slots/slot-1/close") && method === "POST") {
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
      <MemoryRouter>
        <AdminSlotsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AdminSlotsPage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders slots and visits from the API", async () => {
    vi.stubGlobal("fetch", mockSlotsFetch());
    renderPage();

    expect(await screen.findAllByText(/01 Jul 2026/i)).toHaveLength(2);
    expect(screen.getByText("E25-VISIT-abc")).toBeInTheDocument();
  });

  it("opens the add-slot dialog and submits a new slot", async () => {
    const fetchSpy = mockSlotsFetch();
    vi.stubGlobal("fetch", fetchSpy);
    renderPage();

    await screen.findAllByText(/01 Jul 2026/i);

    await userEvent.click(screen.getByRole("button", { name: /add slot/i }));

    const startInput = screen.getByLabelText(/start/i);
    const endInput = screen.getByLabelText(/end/i);

    await userEvent.type(startInput, "2026-07-02T10:00");
    await userEvent.type(endInput, "2026-07-02T11:00");

    await userEvent.click(screen.getByRole("button", { name: /^add slot$/i }));

    await waitFor(() => {
      const postCall = fetchSpy.mock.calls.find(([, init]) => init?.method === "POST");
      expect(postCall).toBeTruthy();
      expect(String(postCall![0])).toContain("/api/v1/admin/slots");
    });
  });

  it("closes an open slot", async () => {
    const fetchSpy = mockSlotsFetch();
    vi.stubGlobal("fetch", fetchSpy);
    renderPage();

    await screen.findAllByText(/01 Jul 2026/i);

    await userEvent.click(screen.getByRole("button", { name: /close/i }));

    await waitFor(() => {
      const closeCall = fetchSpy.mock.calls.find(
        ([url, init]) => String(url).includes("/close") && init?.method === "POST",
      );
      expect(closeCall).toBeTruthy();
    });
  });
});

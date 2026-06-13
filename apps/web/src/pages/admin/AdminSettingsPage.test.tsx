import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { AdminSettingsPage } from "./AdminSettingsPage";

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

const defaultSettings = {
  depositPesewas: 500_00,
  whatsappNumber: "+233200000000",
  visitLocation: "Osu, Accra",
  deliveryRates: [{ area: "Accra", ratePesewas: 1000 }],
};

function mockSettingsFetch() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? "GET";

    if (url.includes("/api/v1/settings") && method === "GET") {
      return jsonResponse(200, { data: defaultSettings });
    }
    if (url.includes("/api/v1/admin/settings") && method === "PUT") {
      return jsonResponse(200, { data: await parseBody(init) });
    }

    return jsonResponse(404, { error: { code: "not_found", message: "no route" } });
  });
}

async function parseBody(init?: RequestInit): Promise<unknown> {
  if (!init?.body) return {};
  return JSON.parse(init.body as string);
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AdminSettingsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AdminSettingsPage", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders settings from the API", async () => {
    vi.stubGlobal("fetch", mockSettingsFetch());
    renderPage();

    expect(await screen.findByDisplayValue("500.00")).toBeInTheDocument();
    expect(screen.getByDisplayValue("+233200000000")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Osu, Accra")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Accra")).toBeInTheDocument();
  });

  it("shows a skeleton while loading", () => {
    vi.stubGlobal("fetch", () => new Promise(() => {}));
    renderPage();

    const skeletons = screen.getAllByText((_, element) =>
      Boolean(element?.className.includes("MuiSkeleton-root")),
    );
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("saves updated settings including delivery rates", async () => {
    const fetchSpy = mockSettingsFetch();
    vi.stubGlobal("fetch", fetchSpy);
    renderPage();

    await screen.findByDisplayValue("500.00");

    await userEvent.clear(screen.getByLabelText(/home-visit deposit/i));
    await userEvent.type(screen.getByLabelText(/home-visit deposit/i), "600");

    await userEvent.clear(screen.getByLabelText(/whatsapp number/i));
    await userEvent.type(screen.getByLabelText(/whatsapp number/i), "+233201111111");

    await userEvent.click(screen.getByRole("button", { name: /add area/i }));
    const areaInputs = screen.getAllByLabelText(/area/i);
    await userEvent.type(areaInputs[areaInputs.length - 1], "Tema");

    const rateInputs = screen.getAllByLabelText(/rate/i);
    await userEvent.type(rateInputs[rateInputs.length - 1], "25.00");

    await userEvent.click(screen.getByRole("button", { name: /save settings/i }));

    await waitFor(() => {
      expect(screen.getByText(/settings saved/i)).toBeInTheDocument();
    });

    const putCall = fetchSpy.mock.calls.find(([, init]) => init?.method === "PUT");
    expect(putCall).toBeTruthy();
    expect(String(putCall![0])).toContain("/api/v1/admin/settings");

    const body = await parseBody(putCall![1]);
    expect(body).toMatchObject({
      depositPesewas: 600_00,
      whatsappNumber: "+233201111111",
      deliveryRates: expect.arrayContaining([{ area: "Tema", ratePesewas: 2500 }]),
    });
  });

  it("blocks save with duplicate delivery areas", async () => {
    vi.stubGlobal("fetch", mockSettingsFetch());
    renderPage();

    await screen.findByDisplayValue("500.00");

    await userEvent.click(screen.getByRole("button", { name: /add area/i }));
    const areaInputs = screen.getAllByLabelText(/area/i);
    await userEvent.type(areaInputs[areaInputs.length - 1], "Accra");

    await userEvent.click(screen.getByRole("button", { name: /save settings/i }));

    expect(await screen.findByText(/listed more than once/i)).toBeInTheDocument();
  });
});

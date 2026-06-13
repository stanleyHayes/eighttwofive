import { describe, it, expect, vi, beforeEach } from "vitest";
import { request, ApiError } from "./api";

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

describe("request", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("defaults Content-Type to application/json", async () => {
    const fetchSpy = vi.fn().mockResolvedValue(jsonResponse(200, { data: {} }));
    vi.stubGlobal("fetch", fetchSpy);

    await request("/api/v1/healthz");

    expect(fetchSpy).toHaveBeenCalledWith(
      "/api/v1/healthz",
      expect.objectContaining({
        credentials: "include",
        headers: { "Content-Type": "application/json" },
      }),
    );
  });

  it("lets caller-provided Content-Type override the default", async () => {
    const fetchSpy = vi.fn().mockResolvedValue(jsonResponse(200, { data: {} }));
    vi.stubGlobal("fetch", fetchSpy);

    await request("/api/v1/uploads/sign", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
    });

    expect(fetchSpy).toHaveBeenCalledWith(
      "/api/v1/uploads/sign",
      expect.objectContaining({
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
      }),
    );
  });

  it("merges extra caller headers without touching Content-Type", async () => {
    const fetchSpy = vi.fn().mockResolvedValue(jsonResponse(200, { data: {} }));
    vi.stubGlobal("fetch", fetchSpy);

    await request("/api/v1/healthz", {
      headers: { "X-Custom": "value" },
    });

    expect(fetchSpy).toHaveBeenCalledWith(
      "/api/v1/healthz",
      expect.objectContaining({
        headers: { "Content-Type": "application/json", "X-Custom": "value" },
      }),
    );
  });

  it("throws ApiError on envelope errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        jsonResponse(422, { error: { code: "invalid_input", message: "bad" } }),
      ),
    );

    await expect(request("/api/v1/waitlist", { method: "POST", body: "{}" })).rejects.toThrow(
      ApiError,
    );
  });
});

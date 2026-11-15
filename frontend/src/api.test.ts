import { describe, expect, it, vi, afterEach } from "vitest";
import { api, safeSource, makeSettingsPayload } from "./api";
afterEach(() => vi.unstubAllGlobals());
describe("same-origin API client", () => {
  it("sends cookie credentials and JSON mutations", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue({ ok: true, json: async () => ({ id: "1" }) });
    vi.stubGlobal("fetch", fetcher);
    await api("/requests", {
      method: "POST",
      body: JSON.stringify({ confirmed: true }),
    });
    expect(fetcher).toHaveBeenCalledWith(
      "/api/requests",
      expect.objectContaining({
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        method: "POST",
      }),
    );
  });
  it("does not render arbitrary provider error bodies", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => ({ error: "SECRET_PROVIDER_KEY" }),
      }),
    );
    await expect(api("/chat")).rejects.toThrow("Сервис временно недоступен");
  });
  it("keeps unauthorized errors distinguishable", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 401 }),
    );
    await expect(api("/me")).rejects.toMatchObject({ status: 401 });
  });
});
describe("safe links and secret configuration", () => {
  it("accepts knowledge anchors and rejects script URLs", () => {
    expect(safeSource("/knowledge/git#branches")).toBe(
      "/knowledge/git#branches",
    );
    expect(safeSource("javascript:alert(1)")).toBe("#");
    expect(safeSource("https://docs.example.org/a")).toBe(
      "https://docs.example.org/a",
    );
  });
  it("blank secrets preserve existing stored keys and flags are explicit", () => {
    expect(
      makeSettingsPayload({
        routerai_model: "model",
        telegram_token: "",
        routerai_api_key: "",
        clear_telegram_token: true,
      }),
    ).toEqual({ routerai_model: "model", clear_telegram_token: true });
  });
});

describe("malformed source links", () => {
  it("rejects protocol-relative and backslash origin escapes", () => {
    expect(safeSource("//untrusted.example/path")).toBe("#");
    expect(safeSource("/\\untrusted.example/path")).toBe("#");
  });
});

describe("connection failures", () => {
  it("explains offline failure without exposing transport details", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new Error("private transport detail")),
    );
    await expect(api("/chat")).rejects.toThrow("Не удалось подключиться");
  });
  it("supports successful empty responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: true, status: 204 }),
    );
    await expect(
      api("/auth/logout", { method: "POST" }),
    ).resolves.toBeUndefined();
  });
});

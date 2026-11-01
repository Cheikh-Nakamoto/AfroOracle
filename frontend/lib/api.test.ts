import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, apiFetch, latestOnly, resolveApiUrl, singleFlight } from "./api";

const json = (status: number, body: unknown) => new Response(JSON.stringify(body), { status });
afterEach(() => vi.unstubAllGlobals());

describe("apiFetch", () => {
  it("returns data on 200 and sends bearer", async () => {
    const f = vi.fn().mockResolvedValue(json(200, { a: 1 }));
    vi.stubGlobal("fetch", f);
    const r = await apiFetch<{ a: number }>("http://x/", "/api/state", { token: "t" });
    expect(r.data.a).toBe(1);
    expect(f.mock.calls[0][0]).toBe("http://x/api/state");
    expect(f.mock.calls[0][1].headers.Authorization).toBe("Bearer t");
  });
  it("throws ApiError with backend message on 500/401", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(json(401, { error: "unauthorized" })));
    await expect(apiFetch("http://x", "/p")).rejects.toMatchObject({ status: 401, message: "unauthorized" });
  });
  it("only allowed statuses are results", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(json(422, { error: "quorum" })));
    const r = await apiFetch<{ error: string }>("http://x", "/p", { okStatuses: [422] });
    expect(r.status).toBe(422);
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(json(500, { error: "boom" })));
    await expect(apiFetch("http://x", "/p", { okStatuses: [422] })).rejects.toBeInstanceOf(ApiError);
  });
  it("network failure becomes ApiError status 0", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("fail")));
    await expect(apiFetch("http://x", "/p")).rejects.toMatchObject({ status: 0 });
  });
  it("non-JSON error body falls back to HTTP status", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("<html>", { status: 502 })));
    await expect(apiFetch("http://x", "/p")).rejects.toMatchObject({ message: "Erreur HTTP 502" });
  });
});

describe("resolveApiUrl", () => {
  it("prefers configured, trims slash", () => expect(resolveApiUrl("https://api.example.com/")).toBe("https://api.example.com"));
  it("falls back to page host", () => expect(resolveApiUrl("", { protocol: "http:", hostname: "10.0.0.5" })).toBe("http://10.0.0.5:8000"));
});

describe("concurrency helpers", () => {
  it("singleFlight runs once for double clicks", async () => {
    const fn = vi.fn(() => new Promise<number>((r) => setTimeout(() => r(1), 5)));
    const g = singleFlight(fn);
    await Promise.all([g(), g()]);
    expect(fn).toHaveBeenCalledTimes(1);
    await g();
    expect(fn).toHaveBeenCalledTimes(2);
  });
  it("latestOnly aborts the previous signal", () => {
    const l = latestOnly();
    const a = l.next();
    const b = l.next();
    expect(a.aborted).toBe(true);
    expect(b.aborted).toBe(false);
  });
});

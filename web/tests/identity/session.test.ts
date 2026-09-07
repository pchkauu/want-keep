import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiFailure, HttpClient } from "@/api/http";
import { IdentityApi } from "@/features/identity/identity-api";
import type { MemberSession } from "@/features/identity/model";
import { SessionController } from "@/features/identity/session-controller";

const start = Date.parse("2026-09-01T12:00:00Z");
const member: MemberSession = {
  userId: "alex",
  householdId: "family",
  name: "Alex",
  locale: "ru",
  asset: "RUB",
  sessionId: "session-a",
  csrf: "csrf-a",
  authenticatedAt: new Date(start).toISOString(),
  expiresAt: new Date(start + 12 * 3600_000).toISOString(),
  idleExpiresAt: new Date(start + 1800_000).toISOString(),
};
afterEach(() => {
  vi.clearAllTimers();
  vi.useRealTimers();
});

describe("identity session boundaries", () => {
  it("expires exactly at the idle deadline and retains only the same-member draft identity", () => {
    vi.useFakeTimers();
    vi.setSystemTime(start);
    const session = new SessionController(new IdentityApi(new HttpClient()));
    session.accept(member);
    vi.advanceTimersByTime(1799_999);
    expect(session.snapshot().status).toBe("active");
    vi.advanceTimersByTime(1);
    expect(session.snapshot()).toMatchObject({
      status: "expired",
      member: { userId: "alex" },
    });
  });
  it("uses the earlier absolute deadline", () => {
    vi.useFakeTimers();
    vi.setSystemTime(start);
    const session = new SessionController(new IdentityApi(new HttpClient()));
    session.accept({
      ...member,
      expiresAt: new Date(start + 10).toISOString(),
    });
    vi.advanceTimersByTime(10);
    expect(session.snapshot().status).toBe("expired");
  });
  it("reconciles local expiry with a live shared cookie without renewing idle time", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(start);
    const request = vi
      .fn<typeof fetch>()
      .mockResolvedValue(new Response(null, { status: 204 }));
    const api = new IdentityApi(new HttpClient(request));
    const session = new SessionController(api);
    session.accept(member);
    vi.advanceTimersByTime(1800_000);
    expect(session.snapshot().status).toBe("expired");
    const renewed = {
      ...member,
      idleExpiresAt: new Date(start + 3500_000).toISOString(),
    };
    vi.spyOn(api, "me").mockResolvedValue(renewed);
    vi.spyOn(api, "activity");
    await session.verify();
    expect(session.snapshot()).toEqual({ status: "active", member: renewed });
    expect(api.activity).not.toHaveBeenCalled();
    await api.logout();
    expect(request).toHaveBeenCalledWith(
      "/api/v1/auth/logout",
      expect.objectContaining({
        headers: expect.objectContaining({ "X-CSRF-Token": "csrf-a" }),
      }),
    );
  });
  it("rejects a result issued before invalidation and clears drafts on another identity", () => {
    vi.useFakeTimers();
    vi.setSystemTime(start);
    const session = new SessionController(new IdentityApi(new HttpClient()));
    session.accept(member);
    const ticket = session.ticket();
    session.expire();
    expect(() => session.accept(member, ticket)).toThrow("session_changed");
    session.onIdentityChange = vi.fn();
    session.accept({
      ...member,
      userId: "sam",
      sessionId: "session-b",
      csrf: "csrf-b",
    });
    expect(session.onIdentityChange).toHaveBeenCalledOnce();
  });
  it("preserves an ongoing ceremony ticket when a same-session read updates the deadline", () => {
    vi.useFakeTimers();
    vi.setSystemTime(start);
    const session = new SessionController(new IdentityApi(new HttpClient()));
    session.accept(member);
    const ticket = session.ticket();
    session.accept({
      ...member,
      idleExpiresAt: new Date(start + 1900_000).toISOString(),
    });
    expect(session.current(ticket)).toBe(true);
  });
  it("does not send activity more than once per minute and cannot revive an expired session", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(start);
    const api = new IdentityApi(new HttpClient());
    vi.spyOn(api, "activity").mockResolvedValue(undefined);
    vi.spyOn(api, "me").mockResolvedValue(member);
    const session = new SessionController(api);
    session.accept(member);
    await session.activity();
    await session.activity();
    expect(api.activity).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(60_000);
    await session.activity();
    expect(api.activity).toHaveBeenCalledTimes(2);
    vi.advanceTimersByTime(1800_000);
    await session.activity();
    expect(api.activity).toHaveBeenCalledTimes(2);
  });
  it("does not claim logout after losing its response", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(start);
    const api = new IdentityApi(new HttpClient());
    vi.spyOn(api, "logout").mockRejectedValue(
      new ApiFailure("network_unconfirmed"),
    );
    const session = new SessionController(api);
    session.accept(member);
    session.onIdentityChange = vi.fn();
    await expect(session.signOut()).rejects.toThrow("network_unconfirmed");
    expect(session.snapshot()).toEqual({
      status: "unavailable",
      logoutUnconfirmed: true,
    });
    expect(session.onIdentityChange).toHaveBeenCalledOnce();
  });
});

describe("same-origin HTTP boundary", () => {
  it("uses cookies, no-store, CSRF and the caller command key without repeating a mutation", async () => {
    const request = vi
      .fn<typeof fetch>()
      .mockRejectedValue(new Error("lost response"));
    const http = new HttpClient(request);
    http.bind("csrf");
    await expect(
      http.json("/accounts", "POST", { name: "cash" }, "command-id"),
    ).rejects.toThrow("network_unconfirmed");
    expect(request).toHaveBeenCalledOnce();
    expect(request).toHaveBeenCalledWith(
      "/api/v1/accounts",
      expect.objectContaining({
        method: "POST",
        credentials: "same-origin",
        cache: "no-store",
        redirect: "error",
        headers: expect.objectContaining({
          "X-CSRF-Token": "csrf",
          "Idempotency-Key": "command-id",
        }),
      }),
    );
  });
  it("discards a late response from a previous session", async () => {
    let resolve!: (value: Response) => void;
    const http = new HttpClient(
      vi.fn<typeof fetch>().mockImplementation(
        () =>
          new Promise((r) => {
            resolve = r;
          }),
      ),
    );
    http.bind("old");
    const response = http.json("/household");
    http.bind("new");
    resolve(Response.json({ name: "old household" }));
    await expect(response).rejects.toThrow("session_changed");
  });
  it("rejects a malformed success and preserves typed safe errors", async () => {
    const request = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(new Response("not json"))
      .mockResolvedValueOnce(
        Response.json(
          { code: "invitation_expired", message: "do not show raw text" },
          { status: 409 },
        ),
      );
    const http = new HttpClient(request);
    await expect(http.json("/me")).rejects.toThrow("invalid_response");
    await expect(
      http.json("/invitations/preview", "POST", {}),
    ).rejects.toMatchObject({
      code: "invitation_expired",
      status: 409,
      message: "invitation_expired",
    });
  });
});

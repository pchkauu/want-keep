import { describe, expect, it, vi } from "vitest";
import { ApiFailure, HttpClient } from "@/api/http";
import { AccountsApi } from "@/features/accounts/accounts-api";
import { CashAccount } from "@/features/accounts/cash-account";
import { CashController } from "@/features/accounts/cash-controller";

const draft = {
  ...CashAccount.empty(),
  name: "Cash",
  amount: "100.000000000000000001",
  date: "2024-02-29",
};
const account = {
  id: "account",
  name: "Cash",
  asset: "RUB",
  amount: draft.amount,
  partial: false,
  stale: false,
};
function fixture() {
  const api = new AccountsApi(new HttpClient());
  vi.spyOn(api, "recent").mockResolvedValue({ items: [], cursor: undefined });
  vi.spyOn(api, "read").mockResolvedValue(account);
  const controller = new CashController(api, "alex", () => "original-key");
  controller.draft(draft);
  return { api, controller };
}
describe("cash account precision and command recovery", () => {
  it("accepts all supported assets without changing fractional digits", () => {
    for (const asset of ["RUB", "USD", "USDT", "USDC", "BTC", "ETH"])
      expect(() => CashAccount.validate({ ...draft, asset })).not.toThrow();
    expect(
      CashAccount.display("12345678901234567890.000000000000000001", "ru"),
    ).toBe(
      "12\u202f345\u202f678\u202f901\u202f234\u202f567\u202f890,000000000000000001",
    );
    expect(() =>
      CashAccount.validate({ ...draft, amount: "9".repeat(256) }),
    ).not.toThrow();
  });
  it("rejects impossible dates and malformed or overlength money without truncating", () => {
    for (const amount of [
      "-1",
      "1e3",
      "NaN",
      "1.",
      ".1",
      " 1",
      "1,5",
      "9".repeat(257),
    ])
      expect(() => CashAccount.validate({ ...draft, amount })).toThrow();
    for (const date of ["2023-02-29", "2026-04-31", "0000-01-01", null])
      expect(() => CashAccount.validate({ ...draft, date })).toThrow();
  });
  it("reads a committed resource after the creation response is lost", async () => {
    const { api, controller } = fixture();
    vi.spyOn(api, "create").mockRejectedValue(
      new ApiFailure("network_unconfirmed"),
    );
    vi.spyOn(api, "command").mockResolvedValue({
      id: "original-key",
      state: "succeeded",
      accountId: "account",
    });
    await controller.loadRecent();
    await controller.submit();
    await controller.submit();
    expect(controller.snapshot().confirmed).toEqual(account);
    expect(api.create).toHaveBeenCalledOnce();
    expect(api.read).toHaveBeenCalledWith("account");
  });
  it("not_found permits only an explicit retry with the original payload and key", async () => {
    const { api, controller } = fixture();
    vi.spyOn(api, "create")
      .mockRejectedValueOnce(new ApiFailure("network_unconfirmed"))
      .mockResolvedValue({
        id: "original-key",
        state: "succeeded",
        accountId: "account",
      });
    vi.spyOn(api, "command").mockRejectedValue(
      new ApiFailure("not_found", 404),
    );
    await controller.loadRecent();
    await controller.submit();
    controller.draft({ amount: "999" });
    await controller.submit();
    expect(controller.snapshot().draft.amount).toBe(draft.amount);
    expect(api.create).toHaveBeenCalledTimes(1);
    await controller.retry();
    expect(api.create).toHaveBeenNthCalledWith(
      2,
      "original-key",
      draft,
      "alex",
    );
    expect(controller.snapshot().confirmed).toEqual(account);
  });
  it("does not create another account while unresolved recent commands exist", async () => {
    const { api, controller } = fixture();
    vi.mocked(api.recent).mockResolvedValue({
      items: [{ id: "pending-key", state: "pending" }],
      cursor: undefined,
    });
    vi.spyOn(api, "create");
    await controller.loadRecent();
    await controller.submit();
    expect(api.create).not.toHaveBeenCalled();
  });
  it("replays durable pending with the same immutable request, including after another lost response", async () => {
    const { api, controller } = fixture();
    vi.spyOn(api, "create")
      .mockRejectedValueOnce(new ApiFailure("network_unconfirmed"))
      .mockRejectedValueOnce(new ApiFailure("network_unconfirmed"))
      .mockResolvedValue({
        id: "original-key",
        state: "succeeded",
        accountId: "account",
      });
    vi.spyOn(api, "command").mockResolvedValue({
      id: "original-key",
      state: "pending",
    });
    await controller.loadRecent();
    await controller.submit();
    expect(controller.snapshot().retryOriginal).toBe(true);
    controller.draft({ amount: "999" });
    await controller.retry();
    expect(controller.snapshot().retryOriginal).toBe(true);
    await controller.retry();
    expect(api.create).toHaveBeenCalledTimes(3);
    for (const call of vi.mocked(api.create).mock.calls)
      expect(call).toEqual(["original-key", draft, "alex"]);
    expect(controller.snapshot().confirmed).toEqual(account);
  });
  it("reconstructs a recent pending request without replacing its key and permits correcting a rejected mismatch", async () => {
    const { api, controller } = fixture();
    vi.mocked(api.recent).mockResolvedValue({
      items: [{ id: "saved-key", state: "pending" }],
      cursor: undefined,
    });
    vi.spyOn(api, "create")
      .mockRejectedValueOnce(new ApiFailure("duplicate_command", 409))
      .mockResolvedValue({
        id: "saved-key",
        state: "succeeded",
        accountId: "account",
      });
    await controller.loadRecent();
    controller.restore("unrelated");
    expect(controller.snapshot().recoveryId).toBeUndefined();
    controller.restore("saved-key");
    controller.draft({ ...draft, amount: "999" });
    await controller.submit();
    expect(controller.snapshot().error?.code).toBe("duplicate_command");
    expect(controller.snapshot().creation).toBeUndefined();
    controller.draft(draft);
    await controller.submit();
    expect(api.create).toHaveBeenNthCalledWith(2, "saved-key", draft, "alex");
    expect(controller.snapshot().confirmed).toEqual(account);
  });
  it("clears drafts and ignores in-flight results when the actor changes", async () => {
    const { api, controller } = fixture();
    let resolve!: (value: {
      id: string;
      state: "succeeded";
      accountId: string;
    }) => void;
    vi.spyOn(api, "create").mockImplementation(
      () =>
        new Promise((r) => {
          resolve = r;
        }),
    );
    await controller.loadRecent();
    const pending = controller.submit();
    controller.clear();
    resolve({ id: "original-key", state: "succeeded", accountId: "account" });
    await pending;
    expect(controller.snapshot().draft).toEqual(CashAccount.empty());
    expect(controller.snapshot().confirmed).toBeUndefined();
    expect(api.read).not.toHaveBeenCalled();
  });
});

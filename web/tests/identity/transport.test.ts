import { describe, expect, it, vi } from "vitest";
import { WebAuthn } from "@/features/identity/webauthn";
import { LocaleController } from "@/locales/locale";
import { en, ru } from "@/locales/messages";
import { RouteContext } from "@/navigation/route-context";

describe("identity browser transport", () => {
  it("cannot start a late native prompt after leaving the ceremony owner", async () => {
    const auth = new WebAuthn();
    auth.close();
    await expect(
      auth.login({} as Parameters<WebAuthn["login"]>[0]),
    ).rejects.toThrow("passkey_cancelled");
    await expect(
      auth.register({} as Parameters<WebAuthn["register"]>[0]),
    ).rejects.toThrow("passkey_cancelled");
  });
  it("round-trips binary WebAuthn values using base64url", () => {
    const bytes = new Uint8Array([0, 1, 127, 128, 254, 255]);
    expect(WebAuthn.decode(WebAuthn.encode(bytes.buffer))).toEqual(bytes);
    for (const bad of ["", "a", "a=", "a+b/", " aaaa", "Zh"])
      expect(() => WebAuthn.decode(bad)).toThrow();
  });
  it("removes invitation fragments before requests without storing the secret in history", () => {
    const history = { state: null, replaceState: vi.fn() };
    const secret = "a".repeat(43);
    expect(
      RouteContext.takeInvitation(
        { pathname: "/invite", hash: `#${secret}`, search: "" },
        history,
      ),
    ).toBe(secret);
    expect(history.replaceState).toHaveBeenCalledWith(null, "", "/invite");
    expect(
      RouteContext.takeInvitation(
        { pathname: "/invite", hash: "#wrong", search: "" },
        history,
      ),
    ).toBe("");
  });
  it("allows only registered internal routes and non-secret context", () => {
    expect(RouteContext.internal("/onboarding?currency=RUB&token=secret")).toBe(
      "/onboarding?currency=RUB",
    );
    for (const path of [
      "https://attacker.test",
      "//attacker.test",
      "/\\attacker.test",
      "/login?returnTo=bad",
      "/api/v1/me",
      "/unknown",
    ])
      expect(RouteContext.internal(path)).toBeUndefined();
  });
  it("keeps RU/EN coverage and explicit browser selection across profile loading", () => {
    expect(Object.keys(en).sort()).toEqual(Object.keys(ru).sort());
    const storage = {
      getItem: vi.fn().mockReturnValue("en"),
      setItem: vi.fn(),
    };
    const locale = new LocaleController(storage, "ru");
    locale.profile("ru");
    expect(locale.snapshot()).toBe("en");
    locale.select("ru");
    expect(storage.setItem).toHaveBeenCalledWith("want-keep.locale", "ru");
    const fallback = new LocaleController(
      {
        getItem: () => {
          throw Error();
        },
        setItem: () => {
          throw Error();
        },
      },
      "fr",
    );
    expect(fallback.snapshot()).toBe("ru");
    fallback.select("en");
    expect(fallback.snapshot()).toBe("en");
  });
});

import { createContext, useContext } from "react";
import { HttpClient } from "@/api/http";
import { AccountsApi, CashController } from "@/features/accounts";
import { IdentityApi, SessionController } from "@/features/identity";
import { LocaleController } from "@/locales/locale";

export class ApplicationServices {
  readonly http = new HttpClient();
  readonly identity = new SessionController(new IdentityApi(this.http));
  readonly accounts = new AccountsApi(this.http);
  readonly locale: LocaleController;
  private cash?: CashController;
  returnTo?: { userId: string; path: string };
  invitationToken: string;
  constructor(invitationToken = "") {
    this.invitationToken = invitationToken;
    let storage: Storage | undefined;
    try {
      if (typeof window !== "undefined") storage = window.localStorage;
    } catch {
      /* Browser privacy settings may disable storage. */
    }
    this.locale = new LocaleController(
      storage,
      typeof navigator === "undefined" ? "ru" : navigator.language,
    );
    this.identity.onIdentityChange = () => {
      this.cash?.clear();
      this.cash = undefined;
      this.returnTo = undefined;
    };
  }
  consumeInvitation = () => {
    this.invitationToken = "";
  };
  rememberReturn(userId: string, path: string) {
    this.returnTo = { userId, path };
  }
  cashAccount(userId: string) {
    if (!this.cash || this.cash.userId !== userId) {
      this.cash?.clear();
      this.cash = new CashController(this.accounts, userId);
    }
    return this.cash;
  }
}
export const ServicesContext = createContext<ApplicationServices | null>(null);
export function useServices() {
  const services = useContext(ServicesContext);
  if (!services) throw new Error("Application services are required");
  return services;
}

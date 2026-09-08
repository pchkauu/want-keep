import { ApiFailure } from "@/api/http";
import type { HouseholdMember, HouseholdView } from "@/features/household";

export type CashDraft = {
  name: string;
  asset: string;
  ownership: "personal" | "household";
  date: string | null;
  dateDraft?: string;
  amount: string;
};
export type AccountSummary = {
  id: string;
  name: string;
  asset: string;
  amount?: string;
  partial: boolean;
  stale: boolean;
  revision: number;
  ownership:
    | { scope: "household"; householdId: string }
    | { scope: "personal"; householdId: string; personalOwnerId: string };
  externalAccountOwnerId?: string;
};
export type AccountGroup = {
  key: string;
  scope: "household" | "personal";
  owner?: HouseholdMember;
  items: readonly AccountSummary[];
};
export type Creation = {
  id: string;
  state: "pending" | "succeeded" | "failed";
  accountId?: string;
};

export class CashAccount {
  static empty(): CashDraft {
    return {
      name: "",
      asset: "RUB",
      ownership: "personal",
      date: null,
      amount: "",
    };
  }
  static validate(draft: CashDraft) {
    const date = draft.date;
    if (
      !draft.name.trim() ||
      draft.name.length > 2000 ||
      !date ||
      !/^\d{4}-\d{2}-\d{2}$/.test(date) ||
      !/^(0|[1-9][0-9]*)(\.[0-9]+)?$/.test(draft.amount) ||
      draft.amount.length > 256 ||
      !["RUB", "USD", "USDT", "USDC", "BTC", "ETH"].includes(draft.asset)
    )
      throw new ApiFailure("cashInvalid");
    const parsed = new Date(`${date}T00:00:00Z`);
    if (
      date.startsWith("0000") ||
      !Number.isFinite(parsed.getTime()) ||
      parsed.toISOString().slice(0, 10) !== date
    )
      throw new ApiFailure("cashInvalid");
  }
  static display(amount: string, locale: "ru" | "en") {
    const [integer, fraction] = amount.split(".");
    const grouped = integer.replace(
      /\B(?=(\d{3})+(?!\d))/g,
      locale === "ru" ? "\u202f" : ",",
    );
    return (
      grouped +
      (fraction === undefined
        ? ""
        : `${locale === "ru" ? "," : "."}${fraction}`)
    );
  }
}

export class AccountCollection {
  static visible(items: readonly AccountSummary[], view: HouseholdView) {
    if (view.view === "household") return [...items];
    return items.filter(
      (account) =>
        account.ownership.scope === "household" ||
        account.ownership.personalOwnerId === view.memberId,
    );
  }

  static groups(
    items: readonly AccountSummary[],
    view: HouseholdView,
    members: readonly HouseholdMember[],
  ): AccountGroup[] {
    const visible = this.visible(items, view);
    const groups: AccountGroup[] = [];
    const shared = visible.filter(
      (account) => account.ownership.scope === "household",
    );
    if (shared.length)
      groups.push({
        key: "household",
        scope: "household",
        items: shared,
      });
    const owners = new Set(
      visible.flatMap((account) =>
        account.ownership.scope === "personal"
          ? [account.ownership.personalOwnerId]
          : [],
      ),
    );
    for (const ownerId of owners) {
      groups.push({
        key: `personal:${ownerId}`,
        scope: "personal",
        owner: members.find((member) => member.userId === ownerId),
        items: visible.filter(
          (account) =>
            account.ownership.scope === "personal" &&
            account.ownership.personalOwnerId === ownerId,
        ),
      });
    }
    return groups;
  }
}

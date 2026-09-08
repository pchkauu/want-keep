import { ApiFailure, HttpClient } from "@/api/http";
import { decodeMoney } from "@/api/money";
import type { components } from "@/api/generated/openapi.gen";
import type { AccountSummary, CashDraft, Creation } from "./cash-account";

type Schema = components["schemas"];

export class AccountsApi {
  private readonly http: HttpClient;
  constructor(http: HttpClient) {
    this.http = http;
  }
  private account(dto: Schema["Account"]): AccountSummary {
    const owned = dto.balance.owned;
    if (
      !dto.id ||
      !dto.name ||
      !dto.ownership.householdId ||
      !Number.isSafeInteger(dto.revision) ||
      dto.revision < 1 ||
      (dto.ownership.scope === "personal" && !dto.ownership.personalOwnerId) ||
      !["household", "personal"].includes(dto.ownership.scope)
    )
      throw new ApiFailure("invalid_response");
    return {
      id: dto.id,
      name: dto.name,
      asset: dto.asset,
      revision: dto.revision,
      ownership:
        dto.ownership.scope === "personal"
          ? {
              scope: "personal",
              householdId: dto.ownership.householdId,
              personalOwnerId: dto.ownership.personalOwnerId,
            }
          : {
              scope: "household",
              householdId: dto.ownership.householdId,
            },
      ...(dto.externalAccountOwnerId
        ? { externalAccountOwnerId: dto.externalAccountOwnerId }
        : {}),
      ...(owned.knowledge === "known"
        ? { amount: decodeMoney(owned.value).amount }
        : {}),
      partial: dto.balance.quality.coverage.state !== "complete",
      stale: dto.balance.quality.freshness !== "fresh",
    };
  }
  async list(cursor?: string) {
    const page = await this.http.json<Schema["AccountPage"]>(
      `/accounts?limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
    );
    return {
      items: page.items.map((dto) => this.account(dto)),
      cursor: page.nextCursor,
    };
  }
  async read(id: string) {
    return this.account(
      await this.http.json<Schema["Account"]>(
        `/accounts/${encodeURIComponent(id)}`,
      ),
    );
  }
  private creation(dto: Schema["CommandStatus"]): Creation {
    if (
      !["pending", "succeeded", "failed"].includes(dto.status) ||
      !dto.id ||
      dto.type !== "accounts.create" ||
      (dto.status === "succeeded" &&
        (!dto.result?.id || dto.result.type !== "account"))
    )
      throw new ApiFailure("invalid_response");
    return {
      id: dto.id,
      state: dto.status,
      ...(dto.status === "succeeded" ? { accountId: dto.result.id } : {}),
    };
  }
  async create(id: string, draft: CashDraft, userId: string) {
    const money = decodeMoney({ asset: draft.asset, amount: draft.amount });
    const input: Schema["AccountCreate"] = {
      name: draft.name,
      asset: money.asset,
      product: "cash",
      openingDate: draft.date!,
      openingBalance: money,
      ownership:
        draft.ownership === "personal"
          ? { scope: "personal", personalOwnerId: userId }
          : { scope: "household" },
    };
    return this.creation(
      await this.http.json<Schema["CommandStatus"]>(
        "/accounts",
        "POST",
        input,
        id,
      ),
    );
  }
  async command(id: string): Promise<Creation> {
    try {
      return this.creation(
        await this.http.json<Schema["CommandStatus"]>(
          `/commands/${encodeURIComponent(id)}`,
        ),
      );
    } catch (error) {
      if (!(error instanceof ApiFailure) || error.status !== 410) throw error;
      const outcome = error.outcome as Schema["CommandOutcome"] | undefined;
      if (!outcome || outcome.commandId !== id) throw error;
      if (
        outcome.status === "succeeded" &&
        outcome.result?.type === "account" &&
        outcome.result.id
      )
        return { id, state: "succeeded", accountId: outcome.result.id };
      if (outcome.status === "failed") return { id, state: "failed" };
      throw error;
    }
  }
  async recent(cursor?: string) {
    const page = await this.http.json<Schema["CommandStatusPage"]>(
      `/commands/recent?limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
    );
    return {
      items: page.items
        .filter((x) => x.type === "accounts.create")
        .map((dto) => this.creation(dto)),
      cursor: page.nextCursor,
    };
  }
}

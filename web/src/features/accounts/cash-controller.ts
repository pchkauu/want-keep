import { ApiFailure } from "@/api/http";
import { AccountsApi } from "./accounts-api";
import {
  CashAccount,
  type CashDraft,
  type Creation,
  type AccountSummary,
} from "./cash-account";

export type CashState = {
  draft: CashDraft;
  busy: boolean;
  creation?: Creation;
  confirmed?: AccountSummary;
  error?: ApiFailure;
  retryOriginal: boolean;
  recent: readonly Creation[];
  checkedRecent: boolean;
};

export class CashController {
  private state: CashState = {
    draft: CashAccount.empty(),
    busy: false,
    retryOriginal: false,
    recent: [],
    checkedRecent: false,
  };
  private payload?: CashDraft;
  private listeners = new Set<() => void>();
  private epoch = 0;
  readonly api: AccountsApi;
  readonly userId: string;
  private readonly key: () => string;
  constructor(
    api: AccountsApi,
    userId: string,
    key: () => string = () => crypto.randomUUID(),
  ) {
    this.api = api;
    this.userId = userId;
    this.key = key;
  }
  snapshot = () => this.state;
  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  };
  private update(patch: Partial<CashState>) {
    this.state = { ...this.state, ...patch };
    this.listeners.forEach((listener) => listener());
  }
  draft(patch: Partial<CashDraft>) {
    if (this.state.busy || this.state.creation) return;
    this.update({ draft: { ...this.state.draft, ...patch }, error: undefined });
  }
  reset() {
    if (this.state.busy || this.state.creation?.state === "pending") return;
    this.payload = undefined;
    this.update({
      draft: CashAccount.empty(),
      creation: undefined,
      confirmed: undefined,
      error: undefined,
      retryOriginal: false,
    });
  }
  clear() {
    this.epoch++;
    this.payload = undefined;
    this.update({
      draft: CashAccount.empty(),
      creation: undefined,
      confirmed: undefined,
      error: undefined,
      recent: [],
      checkedRecent: false,
      retryOriginal: false,
      busy: false,
    });
  }
  private async run(action: (epoch: number) => Promise<void>) {
    if (this.state.busy) return;
    const epoch = this.epoch;
    this.update({ busy: true, error: undefined });
    try {
      await action(epoch);
    } catch (error) {
      if (epoch === this.epoch)
        this.update({
          error:
            error instanceof ApiFailure
              ? error
              : new ApiFailure("service_unavailable"),
        });
    } finally {
      if (epoch === this.epoch) this.update({ busy: false });
    }
  }
  loadRecent = () =>
    this.run(async (epoch) => {
      const recent: Creation[] = [];
      let cursor: string | undefined;
      const seen = new Set<string>();
      do {
        const page = await this.api.recent(cursor);
        if (epoch !== this.epoch) return;
        recent.push(...page.items);
        cursor = page.cursor;
        if (cursor && seen.has(cursor))
          throw new ApiFailure("invalid_response");
        if (cursor) seen.add(cursor);
      } while (cursor);
      if (epoch === this.epoch)
        this.update({
          recent: recent.filter((x) => x.state === "pending"),
          checkedRecent: true,
        });
    });
  submit = () =>
    this.run(async (epoch) => {
      if (
        !this.state.checkedRecent ||
        this.state.creation ||
        this.state.recent.some((x) => x.state === "pending")
      )
        return;
      CashAccount.validate(this.state.draft);
      const id = this.key();
      this.payload = { ...this.state.draft };
      this.update({ creation: { id, state: "pending" }, retryOriginal: false });
      try {
        await this.finish(
          await this.api.create(id, this.payload, this.userId),
          epoch,
        );
      } catch (error) {
        if (epoch !== this.epoch) return;
        // Even a timeout before receiving headers may follow a committed creation.
        await this.reconcile(id, epoch);
        if (this.snapshot().creation?.state === "pending") throw error;
      }
    });
  check = (id: string) => this.run((epoch) => this.reconcile(id, epoch));
  retry = () =>
    this.run(async (epoch) => {
      const creation = this.state.creation;
      if (!creation || !this.payload || !this.state.retryOriginal) return;
      this.update({ retryOriginal: false });
      await this.finish(
        await this.api.create(creation.id, this.payload, this.userId),
        epoch,
      );
    });
  private async reconcile(id: string, epoch: number) {
    try {
      await this.finish(await this.api.command(id), epoch);
    } catch (error) {
      if (
        epoch === this.epoch &&
        error instanceof ApiFailure &&
        error.status === 404 &&
        this.state.creation?.id === id
      )
        this.update({ retryOriginal: Boolean(this.payload) });
      throw error;
    }
  }
  private async finish(creation: Creation, epoch: number) {
    if (epoch !== this.epoch) return;
    this.update({ creation, retryOriginal: false });
    if (creation.state === "succeeded" && creation.accountId) {
      const confirmed = await this.api.read(creation.accountId);
      if (epoch === this.epoch)
        this.update({
          confirmed,
          recent: this.state.recent.filter((x) => x.id !== creation.id),
        });
    } else if (creation.state === "failed")
      this.update({
        recent: this.state.recent.filter((x) => x.id !== creation.id),
      });
  }
}

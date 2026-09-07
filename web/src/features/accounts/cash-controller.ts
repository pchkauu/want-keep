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
  recoveryId?: string;
};

export class CashController {
  private state: CashState = {
    draft: CashAccount.empty(),
    busy: false,
    retryOriginal: false,
    recent: [],
    checkedRecent: false,
  };
  private request?: { id: string; draft: CashDraft };
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
    if (
      this.state.busy ||
      this.request ||
      this.state.creation?.state === "pending"
    )
      return;
    this.request = undefined;
    this.update({
      draft: CashAccount.empty(),
      creation: undefined,
      confirmed: undefined,
      error: undefined,
      retryOriginal: false,
      recoveryId: undefined,
    });
  }
  clear() {
    this.epoch++;
    this.request = undefined;
    this.update({
      draft: CashAccount.empty(),
      creation: undefined,
      confirmed: undefined,
      error: undefined,
      recent: [],
      checkedRecent: false,
      retryOriginal: false,
      busy: false,
      recoveryId: undefined,
    });
  }
  restore(id: string) {
    if (this.state.busy || this.request) return;
    if (!this.state.recent.some((x) => x.id === id && x.state === "pending"))
      return;
    this.update({
      creation: undefined,
      confirmed: undefined,
      recoveryId: id,
      draft: CashAccount.empty(),
      error: undefined,
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
        (!this.state.recoveryId &&
          this.state.recent.some((x) => x.state === "pending"))
      )
        return;
      CashAccount.validate(this.state.draft);
      const id = this.state.recoveryId ?? this.key();
      this.request = { id, draft: { ...this.state.draft } };
      this.update({ creation: { id, state: "pending" }, retryOriginal: false });
      await this.executeRequest(epoch);
    });
  check = (id: string) => this.run((epoch) => this.reconcile(id, epoch));
  retry = () =>
    this.run(async (epoch) => {
      if (!this.request || !this.state.retryOriginal) return;
      await this.executeRequest(epoch);
    });
  private async executeRequest(epoch: number) {
    const request = this.request;
    if (!request) return;
    try {
      await this.finish(
        await this.api.create(request.id, request.draft, this.userId),
        epoch,
      );
    } catch (error) {
      if (epoch !== this.epoch) return;
      if (
        error instanceof ApiFailure &&
        error.code === "duplicate_command" &&
        this.state.recoveryId === request.id
      ) {
        this.request = undefined;
        this.update({ creation: undefined, retryOriginal: false });
        throw error;
      }
      // Even a timeout before receiving headers may follow a committed creation.
      await this.reconcile(request.id, epoch);
      if (this.state.creation?.state === "pending") throw error;
    }
  }
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
        this.update({ retryOriginal: this.request?.id === id });
      throw error;
    }
  }
  private async finish(creation: Creation, epoch: number) {
    if (epoch !== this.epoch) return;
    if (this.request && this.request.id !== creation.id) {
      if (creation.state !== "pending")
        this.update({
          recent: this.state.recent.filter((x) => x.id !== creation.id),
        });
      return;
    }
    if (creation.state !== "pending" && this.request?.id === creation.id)
      this.request = undefined;
    this.update({
      creation,
      confirmed: undefined,
      recoveryId:
        creation.state === "pending" ? this.state.recoveryId : undefined,
      retryOriginal:
        creation.state === "pending" && this.request?.id === creation.id,
    });
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

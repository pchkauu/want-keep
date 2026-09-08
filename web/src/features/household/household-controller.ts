import { ApiFailure } from "@/api/http";
import type { MemberSession } from "@/features/identity";
import type { HouseholdApi } from "./household-api";
import type { HouseholdState, Invitation } from "./model";

export class HouseholdController {
  private state: HouseholdState = { status: "idle" };
  private readonly listeners = new Set<() => void>();
  private epoch = 0;
  private pending?: { actorKey: string; promise: Promise<void> };

  readonly api: HouseholdApi;

  constructor(api: HouseholdApi) {
    this.api = api;
  }

  snapshot = () => this.state;
  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  };

  private publish(state: HouseholdState) {
    this.state = state;
    this.listeners.forEach((listener) => listener());
  }

  clear() {
    this.epoch++;
    this.pending = undefined;
    this.publish({ status: "idle" });
  }

  load(member: MemberSession, refresh = false): Promise<void> {
    const actorKey = this.actorKey(member);
    if (!refresh && this.state.actorKey === actorKey) {
      if (this.pending?.actorKey === actorKey) return this.pending.promise;
      if (this.state.status === "ready") return Promise.resolve();
    }
    const epoch = ++this.epoch;
    const previous =
      refresh && this.state.status === "ready" && this.state.household
        ? this.state
        : undefined;
    this.publish(
      previous
        ? { ...previous, actorKey, refreshing: true, error: undefined }
        : { status: "loading", actorKey },
    );
    const request = Promise.all([this.api.read(), this.api.invitations()]).then(
      ([household, invitation]) => {
        if (epoch !== this.epoch || this.state.actorKey !== actorKey) return;
        const actor = household.members.find(
          (candidate) => candidate.userId === member.userId,
        );
        if (
          household.id !== member.householdId ||
          !actor ||
          actor.status !== "active"
        ) {
          this.publish({
            status: "error",
            actorKey,
            error: new ApiFailure("invalid_response"),
          });
          return;
        }
        this.publish({
          status: "ready",
          actorKey,
          household,
          invitation,
          refreshing: false,
        });
      },
      (error) => {
        if (epoch !== this.epoch || this.state.actorKey !== actorKey) return;
        const failure =
          error instanceof ApiFailure
            ? error
            : new ApiFailure("service_unavailable");
        if (previous) {
          this.publish({
            ...previous,
            actorKey,
            refreshing: false,
            error: failure,
          });
          return;
        }
        this.publish({
          status: "error",
          actorKey,
          error: failure,
        });
      },
    );
    const completion = request.finally(() => {
      if (this.pending?.promise === completion) this.pending = undefined;
    });
    this.pending = { actorKey, promise: completion };
    return completion;
  }

  async issueInvitation(member: MemberSession) {
    const actorKey = this.actorKey(member);
    await this.load(member);
    const current = this.state;
    if (
      current.status !== "ready" ||
      current.actorKey !== actorKey ||
      !current.household ||
      !current.invitation ||
      current.error
    )
      throw current.error ?? new ApiFailure("session_changed");
    const epoch = this.epoch;
    const result = await this.api.issue(current.invitation.revision);
    if (
      epoch !== this.epoch ||
      this.state.status !== "ready" ||
      this.state.actorKey !== actorKey
    )
      throw new ApiFailure("network_unconfirmed");
    this.updateInvitation(result.state);
    return result;
  }

  updateInvitation(invitation: Invitation) {
    if (this.state.status !== "ready" || !this.state.household) return;
    this.publish({
      ...this.state,
      invitation,
      refreshing: false,
      error: undefined,
    });
  }

  private actorKey(member: MemberSession) {
    return `${member.householdId}:${member.userId}:${member.sessionId}`;
  }
}

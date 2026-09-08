import { ApiFailure } from "@/api/http";
import type { MemberSession } from "@/features/identity";
import type { HouseholdApi } from "./household-api";
import type { HouseholdState, Invitation } from "./model";

export class HouseholdController {
  private state: HouseholdState = { status: "idle" };
  private readonly listeners = new Set<() => void>();
  private epoch = 0;
  private pending?: { actorKey: string; promise: Promise<void> };

  private readonly api: HouseholdApi;

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
    const invitationRequest = this.api.invitations().then(
      (value) => ({ status: "fulfilled", value }) as const,
      (reason: unknown) => ({ status: "rejected", reason }) as const,
    );
    const request = this.api.read().then(
      async (household) => {
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
          ...(previous?.invitation ? { invitation: previous.invitation } : {}),
          invitationLoadStatus: "loading",
          refreshing: false,
        });
        const invitationResult = await invitationRequest;
        if (epoch !== this.epoch || this.state.actorKey !== actorKey) return;
        if (invitationResult.status === "rejected") {
          this.publish({
            status: "ready",
            actorKey,
            household,
            ...(previous?.invitation
              ? { invitation: previous.invitation }
              : {}),
            invitationLoadStatus: "error",
            invitationError: this.failure(invitationResult.reason),
            refreshing: false,
          });
          return;
        }
        this.publish({
          status: "ready",
          actorKey,
          household,
          invitation: invitationResult.value,
          invitationLoadStatus: "ready",
          refreshing: false,
        });
      },
      (error: unknown) => {
        if (epoch !== this.epoch || this.state.actorKey !== actorKey) return;
        this.publishLoadFailure(actorKey, previous, error);
      },
    );
    const completion = request.finally(() => {
      if (this.pending?.promise === completion) this.pending = undefined;
    });
    this.pending = { actorKey, promise: completion };
    return completion;
  }

  async issueInvitation(member: MemberSession) {
    const current = await this.invitationContext(member);
    const result = await this.api.issue(current.invitation.revision);
    this.applyInvitation(current, result.state);
    return result;
  }

  async revokeInvitation(member: MemberSession) {
    const current = await this.invitationContext(member);
    if (current.invitation.current?.status !== "active")
      throw new ApiFailure("invalid_response");
    const invitation = await this.api.revoke(
      current.invitation.current.id,
      current.invitation.revision,
    );
    this.applyInvitation(current, invitation);
    return invitation;
  }

  private async invitationContext(member: MemberSession) {
    const actorKey = this.actorKey(member);
    await this.load(member);
    const current = this.state;
    if (
      current.status !== "ready" ||
      current.actorKey !== actorKey ||
      !current.household ||
      !current.invitation ||
      current.invitationLoadStatus !== "ready" ||
      current.invitationError ||
      current.error
    )
      throw (
        current.error ??
        current.invitationError ??
        new ApiFailure("session_changed")
      );
    return { actorKey, epoch: this.epoch, invitation: current.invitation };
  }

  private applyInvitation(
    current: { actorKey: string; epoch: number },
    invitation: Invitation,
  ) {
    if (
      current.epoch !== this.epoch ||
      this.state.status !== "ready" ||
      this.state.actorKey !== current.actorKey ||
      !this.state.household
    )
      throw new ApiFailure("network_unconfirmed");
    this.publish({
      ...this.state,
      invitation,
      invitationLoadStatus: "ready",
      invitationError: undefined,
      refreshing: false,
      error: undefined,
    });
  }

  private actorKey(member: MemberSession) {
    return `${member.householdId}:${member.userId}:${member.sessionId}`;
  }

  private publishLoadFailure(
    actorKey: string,
    previous: HouseholdState | undefined,
    error: unknown,
  ) {
    const failure = this.failure(error);
    this.publish(
      previous
        ? { ...previous, actorKey, refreshing: false, error: failure }
        : { status: "error", actorKey, error: failure },
    );
  }

  private failure(error: unknown) {
    return error instanceof ApiFailure
      ? error
      : new ApiFailure("service_unavailable");
  }
}

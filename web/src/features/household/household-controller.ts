import { ApiFailure } from "@/api/http";
import type { MemberSession } from "@/features/identity";
import type { HouseholdApi } from "./household-api";
import type { HouseholdState, Invitation } from "./model";

export class HouseholdController {
  private state: HouseholdState = { status: "idle" };
  private readonly listeners = new Set<() => void>();
  private epoch = 0;

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
    this.publish({ status: "idle" });
  }

  load(member: MemberSession, refresh = false) {
    const actorKey = `${member.householdId}:${member.userId}:${member.sessionId}`;
    if (
      !refresh &&
      this.state.actorKey === actorKey &&
      (this.state.status === "loading" || this.state.status === "ready")
    )
      return;
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
    void Promise.all([this.api.read(), this.api.invitations()]).then(
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
}

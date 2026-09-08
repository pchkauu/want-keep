import { ApiFailure } from "@/api/http";

export type HouseholdView =
  { view: "household" } | { view: "member"; memberId: string };

export type HouseholdMember = {
  membershipId: string;
  userId: string;
  name: string;
  status: "active" | "pending";
};

export type Household = {
  id: string;
  name: string;
  timezone: string;
  members: readonly HouseholdMember[];
  maximum: number;
};

export type Invitation = {
  revision: number;
  current?: {
    id: string;
    expiresAt: string;
    status: "active" | "accepted" | "revoked";
  };
};

export type ReportView =
  { view: "household" } | { view: "member"; memberId: string };

export class HouseholdViewPolicy {
  static fromSearch(search: string): HouseholdView {
    const parameters = new URLSearchParams(search);
    const memberId = parameters.get("member");
    return parameters.get("view") === "member" && memberId
      ? { view: "member", memberId }
      : { view: "household" };
  }

  static normalize(
    requested: HouseholdView,
    members: readonly HouseholdMember[],
  ): HouseholdView {
    if (
      requested.view === "member" &&
      members.some(
        (member) =>
          member.userId === requested.memberId && member.status === "active",
      )
    )
      return requested;
    return { view: "household" };
  }

  static search(view: HouseholdView, current = ""): string {
    const parameters = new URLSearchParams(current);
    parameters.set("view", view.view);
    if (view.view === "member") parameters.set("member", view.memberId);
    else parameters.delete("member");
    return parameters.toString();
  }

  static path(pathname: string, view: HouseholdView): string {
    return `${pathname}?${this.search(view)}`;
  }

  static report(view: HouseholdView): ReportView {
    return view.view === "member"
      ? { view: "member", memberId: view.memberId }
      : { view: "household" };
  }

  static key(view: HouseholdView): string {
    return view.view === "member" ? `member:${view.memberId}` : "household";
  }

  static fromKey(key: string): HouseholdView | undefined {
    if (key === "household") return { view: "household" };
    if (key.startsWith("member:") && key.length > 7)
      return { view: "member", memberId: key.slice(7) };
  }
}

export type HouseholdState = {
  status: "idle" | "loading" | "ready" | "error";
  refreshing?: boolean;
  actorKey?: string;
  household?: Household;
  invitation?: Invitation;
  invitationLoadStatus?: "loading" | "ready" | "error";
  invitationError?: ApiFailure;
  error?: ApiFailure;
};

export class HouseholdStatePolicy {
  static forActor(state: HouseholdState, actorKey: string): HouseholdState {
    return state.actorKey === actorKey
      ? state
      : { status: "loading", actorKey };
  }
}

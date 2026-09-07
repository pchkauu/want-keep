import { ApiFailure } from "@/api/http";
import { IdentityApi } from "./identity-api";
import type { MemberSession } from "./model";

export type SessionState = {
  status:
    | "checking"
    | "active"
    | "anonymous"
    | "expired"
    | "unavailable"
    | "signing_out";
  member?: MemberSession;
  logoutUnconfirmed?: boolean;
};

export class SessionController {
  private state: SessionState = { status: "checking" };
  private listeners = new Set<() => void>();
  private epoch = 0;
  private lastActivity = -Infinity;
  private activityPending = false;
  private probe?: Promise<void>;
  private timer?: ReturnType<typeof setTimeout>;
  private channel?: BroadcastChannel;
  onIdentityChange: () => void = () => {};

  readonly api: IdentityApi;
  private readonly now: () => number;
  constructor(api: IdentityApi, now = Date.now) {
    this.api = api;
    this.now = now;
    api.http.onUnauthorized = () => this.expire();
  }
  snapshot = () => this.state;
  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  };
  ticket() {
    return this.epoch;
  }
  current(ticket: number) {
    return this.epoch === ticket;
  }

  private publish(state: SessionState) {
    this.state = state;
    clearTimeout(this.timer);
    if (state.status === "active" && state.member) {
      const deadline = Math.min(
        Date.parse(state.member.expiresAt),
        Date.parse(state.member.idleExpiresAt),
      );
      this.timer = setTimeout(
        () => this.expire(),
        Math.max(0, deadline - this.now()),
      );
    }
    this.listeners.forEach((listener) => listener());
  }

  accept(member: MemberSession, ticket = this.epoch) {
    if (!this.current(ticket)) throw new ApiFailure("session_changed");
    if (this.state.member && this.state.member.userId !== member.userId)
      this.onIdentityChange();
    if (
      this.state.member?.sessionId !== member.sessionId ||
      this.state.member?.userId !== member.userId
    )
      this.epoch++;
    this.api.http.bind(member.csrf);
    this.publish({ status: "active", member });
  }

  expire() {
    this.epoch++;
    this.api.http.bind("");
    this.publish({ status: "expired", member: this.state.member });
  }

  verify = (): Promise<void> => {
    if (this.probe) return this.probe;
    const ticket = this.epoch;
    const previous = this.state.member;
    this.probe = this.api
      .me()
      .then((member) => {
        if (this.current(ticket)) this.accept(member, ticket);
      })
      .catch((error) => {
        // The HTTP guard may have expired the session while this probe was in flight.
        if (!this.current(ticket)) return;
        if (error instanceof ApiFailure && error.status === 401) {
          this.api.http.bind("");
          this.publish({
            status: previous ? "expired" : "anonymous",
            member: previous,
          });
        } else this.publish({ status: "unavailable", member: previous });
      })
      .finally(() => {
        this.probe = undefined;
      });
    return this.probe;
  };

  async signOut() {
    this.epoch++;
    this.onIdentityChange();
    this.publish({ status: "signing_out" });
    try {
      await this.api.logout();
      this.api.http.bind("");
      this.publish({ status: "anonymous" });
      this.channel?.postMessage("access_changed");
    } catch (error) {
      this.api.http.bind("");
      this.publish({ status: "unavailable", logoutUnconfirmed: true });
      throw error;
    }
  }

  async activity() {
    if (
      this.state.status !== "active" ||
      this.activityPending ||
      this.now() - this.lastActivity < 60_000
    )
      return;
    const member = this.state.member;
    if (
      !member ||
      this.now() >=
        Math.min(Date.parse(member.idleExpiresAt), Date.parse(member.expiresAt))
    ) {
      this.expire();
      return;
    }
    this.activityPending = true;
    this.lastActivity = this.now();
    try {
      await this.api.activity();
      await this.verify();
    } catch {
      /* The current confirmed deadline still closes the view. */
    } finally {
      this.activityPending = false;
    }
  }

  start() {
    const activity = (event: Event) => {
      if (event.isTrusted && document.visibilityState === "visible")
        void this.activity();
    };
    const verify = () => {
      if (document.visibilityState !== "visible") return;
      void this.verify();
    };
    if (typeof BroadcastChannel !== "undefined") {
      this.channel = new BroadcastChannel("want-keep-access");
      this.channel.onmessage = (event) => {
        if (event.data !== "access_changed") return;
        this.expire();
        verify();
      };
    }
    document.addEventListener("pointerdown", activity);
    document.addEventListener("keydown", activity);
    document.addEventListener("visibilitychange", verify);
    window.addEventListener("online", verify);
    window.addEventListener("pageshow", verify);
    void this.verify();
    return () => {
      clearTimeout(this.timer);
      this.channel?.close();
      document.removeEventListener("pointerdown", activity);
      document.removeEventListener("keydown", activity);
      document.removeEventListener("visibilitychange", verify);
      window.removeEventListener("online", verify);
      window.removeEventListener("pageshow", verify);
    };
  }

  announce() {
    this.channel?.postMessage("access_changed");
  }
}

import { describe, expect, it, vi } from "vitest";
import { HttpClient } from "@/api/http";
import { AccountsApi } from "@/features/accounts/accounts-api";
import {
  AccountCollection,
  type AccountSummary,
} from "@/features/accounts/cash-account";
import {
  HouseholdApi,
  HouseholdController,
  HouseholdViewPolicy,
  type Household,
  type HouseholdMember,
} from "@/features/household";
import type { MemberSession } from "@/features/identity";

const members: HouseholdMember[] = [
  {
    membershipId: "membership-a",
    userId: "user-a",
    name: "Andrey",
    status: "active",
  },
  {
    membershipId: "membership-b",
    userId: "user-b",
    name: "Partner",
    status: "active",
  },
];
const household: Household = {
  id: "household-a",
  name: "Family",
  timezone: "Europe/Moscow",
  members,
  maximum: 2,
};
const session = (
  userId = "user-a",
  sessionId = "session-a",
): MemberSession => ({
  userId,
  sessionId,
  householdId: "household-a",
  name: userId === "user-a" ? "Andrey" : "Partner",
  locale: "en",
  asset: "RUB",
  csrf: "csrf-token-value-that-is-long-enough",
  authenticatedAt: "2026-09-08T10:00:00Z",
  expiresAt: "2026-09-08T22:00:00Z",
  idleExpiresAt: "2026-09-08T10:30:00Z",
});

describe("household view policy", () => {
  it("canonicalizes invalid and inactive members to the household view", () => {
    expect(
      HouseholdViewPolicy.normalize(
        HouseholdViewPolicy.fromSearch("?view=member&member=user-b"),
        members,
      ),
    ).toEqual({ view: "member", memberId: "user-b" });
    for (const search of [
      "?view=member&member=unknown",
      "?view=member",
      "?view=other&member=user-a",
    ])
      expect(
        HouseholdViewPolicy.normalize(
          HouseholdViewPolicy.fromSearch(search),
          members,
        ),
      ).toEqual({ view: "household" });
    expect(
      HouseholdViewPolicy.normalize({ view: "member", memberId: "user-b" }, [
        { ...members[1], status: "pending" },
      ]),
    ).toEqual({ view: "household" });
  });

  it("provides the report contract and carries only the view to another section", () => {
    const view = { view: "member", memberId: "user-b" } as const;
    expect(HouseholdViewPolicy.report(view)).toEqual({
      view: "member",
      memberId: "user-b",
    });
    expect(
      HouseholdViewPolicy.search(view, "period=month&view=household"),
    ).toBe("period=month&view=member&member=user-b");
    expect(HouseholdViewPolicy.path("/accounts", view)).toBe(
      "/accounts?view=member&member=user-b",
    );
  });
});

describe("household loading boundary", () => {
  it("reloads household data when the same actor receives a new session", async () => {
    const api = new HouseholdApi(new HttpClient());
    const refreshed = { ...household, name: "Refreshed family" };
    const read = vi
      .spyOn(api, "read")
      .mockResolvedValueOnce(household)
      .mockResolvedValueOnce(refreshed);
    vi.spyOn(api, "invitations")
      .mockResolvedValueOnce({ revision: 1 })
      .mockResolvedValueOnce({ revision: 2 });
    const controller = new HouseholdController(api);
    controller.load(session());
    await vi.waitFor(() =>
      expect(controller.snapshot()).toMatchObject({
        status: "ready",
        actorKey: "household-a:user-a:session-a",
        household,
      }),
    );
    controller.load(session("user-a", "session-new"));
    expect(controller.snapshot()).toEqual({
      status: "loading",
      actorKey: "household-a:user-a:session-new",
    });
    await vi.waitFor(() =>
      expect(controller.snapshot()).toMatchObject({
        status: "ready",
        actorKey: "household-a:user-a:session-new",
        household: refreshed,
        invitation: { revision: 2 },
      }),
    );
    expect(read).toHaveBeenCalledTimes(2);
  });

  it("waits for the replacement session state before issuing an invitation", async () => {
    const api = new HouseholdApi(new HttpClient());
    let finishInvitation!: (value: { revision: number }) => void;
    vi.spyOn(api, "read").mockResolvedValue(household);
    vi.spyOn(api, "invitations")
      .mockResolvedValueOnce({ revision: 1 })
      .mockImplementationOnce(
        () => new Promise((resolve) => (finishInvitation = resolve)),
      );
    const issue = vi.spyOn(api, "issue").mockResolvedValue({
      state: { revision: 3 },
      token: "a".repeat(43),
    });
    const controller = new HouseholdController(api);
    await controller.load(session());

    const replacement = session("user-a", "session-new");
    const loading = controller.load(replacement);
    const result = controller.issueInvitation(replacement);
    expect(controller.load(replacement)).toBe(loading);
    expect(issue).not.toHaveBeenCalled();

    finishInvitation({ revision: 2 });
    await expect(result).resolves.toMatchObject({
      state: { revision: 3 },
    });
    expect(issue).toHaveBeenCalledWith(2);
    expect(controller.snapshot()).toMatchObject({
      status: "ready",
      actorKey: "household-a:user-a:session-new",
      invitation: { revision: 3 },
    });
  });

  it("drops late data from a replaced session", async () => {
    const api = new HouseholdApi(new HttpClient());
    let finishFirst!: (value: Household) => void;
    vi.spyOn(api, "read")
      .mockImplementationOnce(
        () => new Promise((resolve) => (finishFirst = resolve)),
      )
      .mockResolvedValue({ ...household, members: [members[1]] });
    vi.spyOn(api, "invitations")
      .mockResolvedValueOnce({ revision: 1 })
      .mockResolvedValueOnce({ revision: 2 });
    const controller = new HouseholdController(api);
    controller.load(session());
    controller.load(session("user-b", "session-b"));
    finishFirst(household);
    await vi.waitFor(() =>
      expect(controller.snapshot()).toMatchObject({
        status: "ready",
        actorKey: "household-a:user-b:session-b",
        invitation: { revision: 2 },
      }),
    );
    expect(controller.snapshot().household?.members).toEqual([members[1]]);
  });

  it("rejects a household response that does not contain the active actor", async () => {
    const api = new HouseholdApi(new HttpClient());
    vi.spyOn(api, "read").mockResolvedValue({
      ...household,
      members: [{ ...members[0], status: "pending" }],
    });
    vi.spyOn(api, "invitations").mockResolvedValue({ revision: 1 });
    const controller = new HouseholdController(api);
    controller.load(session());
    await vi.waitFor(() =>
      expect(controller.snapshot()).toMatchObject({
        status: "error",
        error: { code: "invalid_response" },
      }),
    );
  });

  it("keeps the last confirmed household visible when a refresh fails", async () => {
    const api = new HouseholdApi(new HttpClient());
    const read = vi
      .spyOn(api, "read")
      .mockResolvedValueOnce(household)
      .mockRejectedValueOnce(new Error("offline"));
    vi.spyOn(api, "invitations").mockResolvedValue({ revision: 1 });
    const controller = new HouseholdController(api);
    controller.load(session());
    await vi.waitFor(() =>
      expect(controller.snapshot()).toMatchObject({
        status: "ready",
        household,
      }),
    );
    controller.load(session(), true);
    expect(controller.snapshot()).toMatchObject({
      status: "ready",
      refreshing: true,
      household,
    });
    await vi.waitFor(() =>
      expect(controller.snapshot()).toMatchObject({
        status: "ready",
        refreshing: false,
        household,
        error: { code: "service_unavailable" },
      }),
    );
    expect(read).toHaveBeenCalledTimes(2);
  });
});

describe("household transport and account presentation", () => {
  it("maps membership status and sends revision-bound invitation changes", async () => {
    const request = vi
      .fn<typeof fetch>()
      .mockImplementation(async (input, init) => {
        const path = String(input);
        if (path.endsWith("/household"))
          return Response.json({
            id: household.id,
            name: household.name,
            timezone: household.timezone,
            maxActiveMembers: 2,
            members: members.map((member) => ({
              membership: {
                id: member.membershipId,
                householdId: household.id,
                userId: member.userId,
                role: "member",
                status: member.status,
              },
              user: { id: member.userId, name: member.name },
            })),
          });
        if (init?.method === "POST")
          return Response.json({
            state: {
              revision: 2,
              current: {
                id: "invite-a",
                expiresAt: "2026-09-09T10:00:00Z",
                status: "active",
              },
            },
            invitationToken: "a".repeat(43),
          });
        return Response.json({ revision: 1 });
      });
    const http = new HttpClient(request);
    http.bind("csrf-token");
    const api = new HouseholdApi(http);
    expect(await api.read()).toEqual(household);
    expect((await api.issue(1)).state.revision).toBe(2);
    expect(request).toHaveBeenLastCalledWith(
      "/api/v1/household/invitations",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ expectedRevision: 1 }),
      }),
    );
  });

  it("rejects duplicate or over-capacity membership responses", async () => {
    const response = (duplicate = false) =>
      Response.json({
        id: household.id,
        name: household.name,
        timezone: household.timezone,
        maxActiveMembers: 1,
        members: members.map((member) => ({
          membership: {
            id: duplicate ? "duplicate" : member.membershipId,
            householdId: household.id,
            userId: member.userId,
            role: "member",
            status: member.status,
          },
          user: { id: member.userId, name: member.name },
        })),
      });
    await expect(
      new HouseholdApi(
        new HttpClient(vi.fn().mockResolvedValue(response())),
      ).read(),
    ).rejects.toMatchObject({ code: "invalid_response" });
    await expect(
      new HouseholdApi(
        new HttpClient(vi.fn().mockResolvedValue(response(true))),
      ).read(),
    ).rejects.toMatchObject({ code: "invalid_response" });
  });

  it("shows all accounts for the household and shared plus selected personal accounts for a member", () => {
    const account = (
      id: string,
      ownership: AccountSummary["ownership"],
    ): AccountSummary => ({
      id,
      name: id,
      asset: "RUB",
      amount: "100",
      partial: false,
      stale: false,
      revision: 1,
      ownership,
    });
    const accounts = [
      account("shared", { scope: "household", householdId: household.id }),
      account("andrey", {
        scope: "personal",
        householdId: household.id,
        personalOwnerId: "user-a",
      }),
      account("partner", {
        scope: "personal",
        householdId: household.id,
        personalOwnerId: "user-b",
      }),
    ];
    expect(
      AccountCollection.visible(accounts, { view: "household" }).map(
        (account) => account.id,
      ),
    ).toEqual(["shared", "andrey", "partner"]);
    expect(
      AccountCollection.visible(accounts, {
        view: "member",
        memberId: "user-b",
      }).map((account) => account.id),
    ).toEqual(["shared", "partner"]);
  });

  it("uses the signed-in actor as personal owner independently of a selected member view", async () => {
    const request = vi.fn<typeof fetch>().mockResolvedValue(
      Response.json({
        id: "command-a",
        type: "accounts.create",
        status: "pending",
        createdAt: "2026-09-08T10:00:00Z",
        updatedAt: "2026-09-08T10:00:00Z",
      }),
    );
    const http = new HttpClient(request);
    http.bind("csrf-token");
    const api = new AccountsApi(http);
    const selectedView = { view: "member", memberId: "user-b" } as const;
    expect(selectedView.memberId).toBe("user-b");
    await api.create(
      "command-a",
      {
        name: "Actor cash",
        asset: "RUB",
        ownership: "personal",
        date: "2026-09-08",
        amount: "100",
      },
      "user-a",
    );
    const body = JSON.parse(
      String(vi.mocked(request).mock.calls[0]?.[1]?.body),
    ) as { ownership: { scope: string; personalOwnerId: string } };
    expect(body.ownership).toEqual({
      scope: "personal",
      personalOwnerId: "user-a",
    });
  });
});

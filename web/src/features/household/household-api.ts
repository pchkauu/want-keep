import { ApiFailure, HttpClient } from "@/api/http";
import type { components } from "@/api/generated/openapi.gen";
import type { Household, Invitation } from "./model";

type Schema = components["schemas"];

export class HouseholdApi {
  private readonly http: HttpClient;

  constructor(http: HttpClient) {
    this.http = http;
  }

  private invitation(dto: Schema["InvitationState"]): Invitation {
    if (
      !Number.isSafeInteger(dto.revision) ||
      dto.revision < 1 ||
      (dto.current &&
        (!dto.current.id ||
          !["active", "accepted", "revoked"].includes(dto.current.status) ||
          !Number.isFinite(Date.parse(dto.current.expiresAt))))
    )
      throw new ApiFailure("invalid_response");
    return {
      revision: dto.revision,
      ...(dto.current
        ? {
            current: {
              id: dto.current.id,
              expiresAt: dto.current.expiresAt,
              status: dto.current.status,
            },
          }
        : {}),
    };
  }

  async read(): Promise<Household> {
    const dto = await this.http.json<Schema["Household"]>("/household");
    const members = Array.isArray(dto.members) ? dto.members : [];
    const membershipIds = new Set(
      members.map((member) => member.membership.id),
    );
    const userIds = new Set(members.map((member) => member.user.id));
    const activeMembers = members.filter(
      (member) => member.membership.status === "active",
    ).length;
    if (
      !dto.id ||
      !dto.name ||
      !dto.timezone ||
      !Array.isArray(dto.members) ||
      !Number.isSafeInteger(dto.maxActiveMembers) ||
      dto.maxActiveMembers < 1 ||
      membershipIds.size !== members.length ||
      userIds.size !== members.length ||
      activeMembers > dto.maxActiveMembers ||
      members.some(
        (member) =>
          !member.membership.id ||
          !member.user.id ||
          !member.user.name ||
          member.membership.householdId !== dto.id ||
          member.membership.userId !== member.user.id ||
          member.membership.role !== "member" ||
          !["active", "pending"].includes(member.membership.status),
      )
    )
      throw new ApiFailure("invalid_response");
    return {
      id: dto.id,
      name: dto.name,
      timezone: dto.timezone,
      maximum: dto.maxActiveMembers,
      members: members.map((member) => ({
        membershipId: member.membership.id,
        userId: member.user.id,
        name: member.user.name,
        status: member.membership.status,
      })),
    };
  }

  async invitations(): Promise<Invitation> {
    return this.invitation(
      await this.http.json<Schema["InvitationState"]>("/household/invitations"),
    );
  }

  async issue(revision: number) {
    const dto = await this.http.json<Schema["InvitationCreated"]>(
      "/household/invitations",
      "POST",
      { expectedRevision: revision },
    );
    if (!dto.invitationToken) throw new ApiFailure("invalid_response");
    return {
      state: this.invitation(dto.state),
      token: dto.invitationToken,
    };
  }

  async revoke(id: string, revision: number) {
    return this.invitation(
      await this.http.json<Schema["InvitationState"]>(
        `/household/invitations/${encodeURIComponent(id)}/revoke`,
        "POST",
        { expectedRevision: revision },
      ),
    );
  }
}

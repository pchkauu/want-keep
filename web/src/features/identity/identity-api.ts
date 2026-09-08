import { ApiFailure, HttpClient } from "@/api/http";
import type { components } from "@/api/generated/openapi.gen";
import {
  assets,
  type InvitationPreview,
  type InviteProfile,
  type MemberSession,
  type Setup,
} from "./model";
import { WebAuthn } from "./webauthn";

type Schema = components["schemas"];

export class IdentityApi {
  readonly http: HttpClient;
  constructor(http: HttpClient) {
    this.http = http;
  }

  static session(dto: Schema["Me"]): MemberSession {
    if (
      !dto ||
      !dto.user?.id ||
      !dto.user.name ||
      !dto.membership?.householdId ||
      dto.membership.userId !== dto.user.id ||
      dto.membership.status !== "active" ||
      !["ru", "en"].includes(dto.preferences?.locale) ||
      !assets.includes(dto.preferences.reportingAsset) ||
      !dto.session?.id ||
      !dto.session.current ||
      !dto.csrfToken ||
      [
        dto.session.authenticatedAt,
        dto.session.expiresAt,
        dto.session.idleExpiresAt,
      ].some((x) => !Number.isFinite(Date.parse(x)))
    )
      throw new ApiFailure("invalid_response");
    return {
      userId: dto.user.id,
      householdId: dto.membership.householdId,
      name: dto.user.name,
      locale: dto.preferences.locale,
      asset: dto.preferences.reportingAsset,
      sessionId: dto.session.id,
      authenticatedAt: dto.session.authenticatedAt,
      expiresAt: dto.session.expiresAt,
      idleExpiresAt: dto.session.idleExpiresAt,
      csrf: dto.csrfToken,
    };
  }

  async me() {
    return IdentityApi.session(await this.http.json<Schema["Me"]>("/me"));
  }
  async login(webauthn: WebAuthn, purpose: "login" | "reauthentication") {
    const options = await this.http.json<Schema["LoginOptions"]>(
      "/auth/login/options",
      "POST",
      { purpose },
    );
    const credential = await webauthn.login(options);
    return IdentityApi.session(
      await this.http.json<Schema["Me"]>("/auth/login/verify", "POST", {
        attemptId: options.attemptId,
        credential,
      }),
    );
  }
  async setup(input: Setup, webauthn: WebAuthn, name: string) {
    const options = await this.http.json<Schema["EnrollmentOptions"]>(
      "/household/bootstrap",
      "POST",
      input,
    );
    return this.enroll(options, webauthn, name);
  }
  async recovery(code: string) {
    const result = await this.http.json<Schema["RecoveryAttempt"]>(
      "/auth/recovery",
      "POST",
      { recoveryCode: code },
    );
    return { token: result.enrollmentToken, expiresAt: result.expiresAt };
  }
  async recoverPasskey(token: string, webauthn: WebAuthn, name: string) {
    const options = await this.http.json<Schema["EnrollmentOptions"]>(
      "/auth/enrollment/options",
      "POST",
      { purpose: "recovery", authorizationToken: token },
    );
    return this.enroll(options, webauthn, name);
  }
  private async enroll(
    options: Schema["EnrollmentOptions"],
    webauthn: WebAuthn,
    name: string,
  ) {
    const credential = await webauthn.register(options);
    return this.enrollment(
      await this.http.json<Schema["InitialEnrollmentResult"]>(
        "/auth/enrollment/verify",
        "POST",
        { attemptId: options.attemptId, credential, name },
      ),
    );
  }
  private enrollment(dto: Schema["InitialEnrollmentResult"]) {
    if (
      !dto?.recoveryCodes ||
      !Array.isArray(dto.recoveryCodes.codes) ||
      dto.recoveryCodes.codes.length !== 10 ||
      dto.recoveryCodes.codes.some((c) => typeof c !== "string" || !c)
    )
      throw new ApiFailure("invalid_response");
    return {
      me: IdentityApi.session(dto.me),
      codes: [...dto.recoveryCodes.codes],
    };
  }
  async preview(token: string): Promise<InvitationPreview> {
    const dto = await this.http.json<Schema["InvitationPreview"]>(
      "/invitations/preview",
      "POST",
      { invitationToken: token },
    );
    return {
      householdName: dto.householdName,
      inviterName: dto.inviterName,
      expiresAt: dto.expiresAt,
    };
  }
  async accept(
    token: string,
    profile: InviteProfile,
    webauthn: WebAuthn,
    name: string,
  ) {
    const options = await this.http.json<Schema["EnrollmentOptions"]>(
      "/auth/enrollment/options",
      "POST",
      { purpose: "invitation", authorizationToken: token, ...profile },
    );
    const credential = await webauthn.register(options);
    return this.enrollment(
      await this.http.json<Schema["InitialEnrollmentResult"]>(
        "/invitations/accept",
        "POST",
        {
          invitationToken: token,
          enrollmentAttemptId: options.attemptId,
          credentialName: name,
          credential,
        },
      ),
    );
  }
  async replaceCodes() {
    const result = (
      await this.http.json<Schema["RecoveryCodes"]>(
        "/security/recovery-codes",
        "POST",
        {},
      )
    ).codes;
    if (
      !Array.isArray(result) ||
      result.length !== 10 ||
      result.some((code) => typeof code !== "string" || !code)
    )
      throw new ApiFailure("invalid_response");
    return result;
  }
  activity() {
    return this.http.json<void>("/auth/session/activity", "POST", {});
  }
  logout() {
    return this.http.json<void>("/auth/logout", "POST");
  }
}

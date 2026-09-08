import { ApiFailure } from "@/api/http";
import type { components } from "@/api/generated/openapi.gen";

type Schema = components["schemas"];

export class WebAuthn {
  private pending?: AbortController;
  private closed = false;

  static decode(value: string): Uint8Array<ArrayBuffer> {
    if (!/^[A-Za-z0-9_-]+$/.test(value) || value.length % 4 === 1)
      throw new ApiFailure("invalid_response");
    const bytes = Uint8Array.from(
      atob(value.replace(/-/g, "+").replace(/_/g, "/")),
      (c) => c.charCodeAt(0),
    );
    if (WebAuthn.encode(bytes.buffer) !== value)
      throw new ApiFailure("invalid_response");
    return bytes;
  }

  static encode(value: ArrayBuffer): string {
    return btoa(String.fromCharCode(...new Uint8Array(value)))
      .replace(/\+/g, "-")
      .replace(/\//g, "_")
      .replace(/=+$/, "");
  }

  cancel() {
    this.pending?.abort();
  }
  close() {
    this.closed = true;
    this.cancel();
  }
  activate() {
    this.closed = false;
  }

  private begin() {
    if (this.closed) throw new ApiFailure("passkey_cancelled");
    if (
      !window.isSecureContext ||
      !window.PublicKeyCredential ||
      !navigator.credentials
    )
      throw new ApiFailure("passkey_unavailable");
    this.cancel();
    this.pending = new AbortController();
    return this.pending.signal;
  }

  async login(
    options: Schema["LoginOptions"],
  ): Promise<Schema["AuthenticationCredential"]> {
    try {
      const signal = this.begin();
      const credential = await navigator.credentials.get({
        signal,
        publicKey: {
          challenge: WebAuthn.decode(options.challenge),
          rpId: options.rpId,
          userVerification: "required",
          timeout: options.timeout,
        },
      });
      if (signal.aborted) throw new ApiFailure("passkey_cancelled");
      if (
        !(credential instanceof PublicKeyCredential) ||
        !(credential.response instanceof AuthenticatorAssertionResponse)
      )
        throw new ApiFailure("passkey_cancelled");
      const r = credential.response;
      return {
        id: credential.id,
        rawId: WebAuthn.encode(credential.rawId),
        type: "public-key",
        clientDataJSON: WebAuthn.encode(r.clientDataJSON),
        authenticatorData: WebAuthn.encode(r.authenticatorData),
        signature: WebAuthn.encode(r.signature),
        ...(r.userHandle ? { userHandle: WebAuthn.encode(r.userHandle) } : {}),
      };
    } catch (error) {
      throw this.failure(error);
    }
  }

  async register(
    options: Schema["EnrollmentOptions"],
  ): Promise<Schema["RegistrationCredential"]> {
    try {
      const signal = this.begin();
      const credential = await navigator.credentials.create({
        signal,
        publicKey: {
          challenge: WebAuthn.decode(options.challenge),
          rp: { id: options.rpId, name: options.rpName },
          user: {
            id: WebAuthn.decode(options.userId),
            name: options.userName,
            displayName: options.userDisplayName,
          },
          pubKeyCredParams: options.algorithms.map((alg) => ({
            alg,
            type: "public-key",
          })),
          excludeCredentials: options.excludeCredentials.map((id) => ({
            id: WebAuthn.decode(id),
            type: "public-key",
          })),
          authenticatorSelection: {
            residentKey: "required",
            userVerification: "required",
          },
          attestation: "none",
          timeout: options.timeout,
        },
      });
      if (signal.aborted) throw new ApiFailure("passkey_cancelled");
      if (
        !(credential instanceof PublicKeyCredential) ||
        !(credential.response instanceof AuthenticatorAttestationResponse)
      )
        throw new ApiFailure("passkey_cancelled");
      return {
        id: credential.id,
        rawId: WebAuthn.encode(credential.rawId),
        type: "public-key",
        clientDataJSON: WebAuthn.encode(credential.response.clientDataJSON),
        attestationObject: WebAuthn.encode(
          credential.response.attestationObject,
        ),
        transports: credential.response
          .getTransports()
          .filter((t): t is "usb" | "nfc" | "ble" | "internal" | "hybrid" =>
            ["usb", "nfc", "ble", "internal", "hybrid"].includes(t),
          ),
      };
    } catch (error) {
      throw this.failure(error);
    }
  }

  private failure(error: unknown): ApiFailure {
    if (error instanceof ApiFailure) return error;
    if (
      error instanceof DOMException &&
      ["NotAllowedError", "AbortError"].includes(error.name)
    )
      return new ApiFailure("passkey_cancelled");
    if (error instanceof DOMException && error.name === "InvalidStateError")
      return new ApiFailure("passkey_exists");
    return new ApiFailure("passkey_failed");
  }
}

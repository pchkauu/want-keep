export class ApiFailure extends Error {
  readonly code: string;
  readonly status: number;
  readonly correlationId?: string;
  readonly outcome?: unknown;

  constructor(
    code: string,
    status = 0,
    correlationId?: string,
    outcome?: unknown,
  ) {
    super(code);
    this.code = code;
    this.status = status;
    this.correlationId = correlationId;
    this.outcome = outcome;
  }
}

export class HttpClient {
  private csrf = "";
  private generation = 0;
  onUnauthorized: () => void = () => {};

  private readonly request: typeof fetch;
  constructor(request: typeof fetch = (...args) => fetch(...args)) {
    this.request = request;
  }

  bind(csrf: string) {
    if (this.csrf === csrf) return;
    this.generation++;
    this.csrf = csrf;
  }

  async json<T>(
    path: string,
    method = "GET",
    body?: unknown,
    key?: string,
  ): Promise<T> {
    if (!path.startsWith("/") || path.startsWith("//") || path.includes("#")) {
      throw new ApiFailure("invalid_request");
    }
    const generation = this.generation;
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 25_000);
    try {
      const response = await this.request(`/api/v1${path}`, {
        method,
        credentials: "same-origin",
        cache: "no-store",
        redirect: "error",
        signal: controller.signal,
        headers: {
          Accept: "application/json",
          ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
          ...(method !== "GET" && this.csrf
            ? { "X-CSRF-Token": this.csrf }
            : {}),
          ...(key ? { "Idempotency-Key": key } : {}),
        },
        ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
      });
      if (generation !== this.generation)
        throw new ApiFailure("session_changed");
      if (response.status === 204) return undefined as T;
      const value: unknown = await response.json().catch(() => null);
      if (generation !== this.generation)
        throw new ApiFailure("session_changed");
      if (!response.ok) {
        if (response.status === 401 && this.csrf) this.onUnauthorized();
        const error =
          value && typeof value === "object"
            ? (value as Record<string, unknown>)
            : {};
        throw new ApiFailure(
          typeof error.code === "string" ? error.code : "service_unavailable",
          response.status,
          typeof error.correlationId === "string"
            ? error.correlationId
            : undefined,
          error.outcome,
        );
      }
      if (!value || typeof value !== "object")
        throw new ApiFailure("invalid_response");
      return value as T;
    } catch (error) {
      if (error instanceof ApiFailure) throw error;
      throw new ApiFailure("network_unconfirmed");
    } finally {
      clearTimeout(timeout);
    }
  }
}

const destinations = new Set([
  "/overview",
  "/onboarding",
  "/accounts",
  "/plan",
  "/analytics",
  "/chat",
  "/connections",
  "/settings",
  "/settings/security",
  "/notifications",
]);
const queryKeys = new Set([
  "month",
  "period",
  "view",
  "member",
  "currency",
  "account",
  "category",
  "query",
  "sort",
]);

export class RouteContext {
  static internal(path: string): string | undefined {
    if (
      !path.startsWith("/") ||
      path.startsWith("//") ||
      path.includes("\\") ||
      [...path].some((c) => c.charCodeAt(0) < 32)
    )
      return;
    const url = new URL(path, "https://want-keep.invalid");
    if (
      url.origin !== "https://want-keep.invalid" ||
      !destinations.has(url.pathname)
    )
      return;
    const query = new URLSearchParams();
    for (const [key, value] of url.searchParams)
      if (queryKeys.has(key) && value.length <= 256) query.set(key, value);
    return url.pathname + (query.size ? `?${query}` : "");
  }
  static takeInvitation(
    location: Pick<Location, "pathname" | "hash" | "search">,
    history: Pick<History, "replaceState" | "state">,
  ): string {
    if (location.pathname !== "/invite" || !location.hash) return "";
    const token = location.hash.slice(1);
    history.replaceState(history.state, "", location.pathname);
    return /^[A-Za-z0-9_-]{43}$/.test(token) ? token : "";
  }
}

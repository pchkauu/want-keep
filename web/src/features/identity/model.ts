import type { Locale } from "@/locales/locale";
export type { Locale } from "@/locales/locale";
export type Asset = "RUB" | "USD" | "USDT" | "USDC" | "BTC" | "ETH";
export const assets: readonly Asset[] = [
  "RUB",
  "USD",
  "USDT",
  "USDC",
  "BTC",
  "ETH",
];

export type MemberSession = {
  userId: string;
  householdId: string;
  name: string;
  locale: Locale;
  asset: Asset;
  sessionId: string;
  authenticatedAt: string;
  expiresAt: string;
  idleExpiresAt: string;
  csrf: string;
};
export type Setup = {
  name: string;
  householdName: string;
  timezone: string;
  locale: Locale;
  reportingAsset: Asset;
  bootstrapToken: string;
};
export type InviteProfile = {
  name: string;
  locale: Locale;
  reportingAsset: Asset;
};
export type InvitationPreview = {
  householdName: string;
  inviterName: string;
  expiresAt: string;
};

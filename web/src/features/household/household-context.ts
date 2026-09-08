import { createContext } from "react";
import type { MemberSession } from "@/features/identity";
import type { HouseholdController } from "./household-controller";
import type { HouseholdState, HouseholdView, ReportView } from "./model";

export type HouseholdContextValue = HouseholdState & {
  actor: MemberSession;
  view: HouseholdView;
  reportView: ReportView;
  select: (view: HouseholdView) => void;
  path: (pathname: string) => string;
  refresh: () => void;
  controller: HouseholdController;
};

export const HouseholdContext = createContext<HouseholdContextValue | null>(
  null,
);

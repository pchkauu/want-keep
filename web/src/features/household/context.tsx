import {
  useEffect,
  useMemo,
  useSyncExternalStore,
  type ReactNode,
} from "react";
import { useLocation, useNavigate } from "react-router";
import type { MemberSession } from "@/features/identity";
import type { HouseholdController } from "./household-controller";
import { HouseholdContext } from "./household-context";
import {
  HouseholdViewPolicy,
  type HouseholdView,
  type ReportView,
} from "./model";

function useHouseholdValue(
  controller: HouseholdController,
  actor: MemberSession,
) {
  const state = useSyncExternalStore(
    controller.subscribe,
    controller.snapshot,
    controller.snapshot,
  );
  const location = useLocation();
  const navigate = useNavigate();
  useEffect(() => {
    controller.load(actor);
  }, [actor, controller]);
  const requested = HouseholdViewPolicy.fromSearch(location.search);
  const view = state.household
    ? HouseholdViewPolicy.normalize(requested, state.household.members)
    : requested;
  const canonicalSearch = state.household
    ? HouseholdViewPolicy.search(view, location.search)
    : location.search.slice(1);
  useEffect(() => {
    if (!state.household || canonicalSearch === location.search.slice(1))
      return;
    void navigate(
      { pathname: location.pathname, search: `?${canonicalSearch}` },
      { replace: true },
    );
  }, [
    canonicalSearch,
    location.pathname,
    location.search,
    navigate,
    state.household,
  ]);
  return useMemo(
    () => ({
      ...state,
      actor,
      view,
      reportView: HouseholdViewPolicy.report(view) as ReportView,
      select(next: HouseholdView) {
        if (!state.household) return;
        const normalized = HouseholdViewPolicy.normalize(
          next,
          state.household.members,
        );
        void navigate(
          {
            pathname: location.pathname,
            search: `?${HouseholdViewPolicy.search(normalized, location.search)}`,
          },
          { replace: false },
        );
      },
      path(pathname: string) {
        return HouseholdViewPolicy.path(pathname, view);
      },
      refresh() {
        controller.load(actor, true);
      },
      controller,
    }),
    [
      actor,
      controller,
      location.pathname,
      location.search,
      navigate,
      state,
      view,
    ],
  );
}

export function HouseholdProvider({
  controller,
  actor,
  children,
}: {
  controller: HouseholdController;
  actor: MemberSession;
  children: ReactNode;
}) {
  return (
    <HouseholdContext.Provider value={useHouseholdValue(controller, actor)}>
      {children}
    </HouseholdContext.Provider>
  );
}

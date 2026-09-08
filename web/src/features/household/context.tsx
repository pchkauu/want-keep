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
  HouseholdStatePolicy,
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
  const actorKey = `${actor.householdId}:${actor.userId}:${actor.sessionId}`;
  const visibleState = HouseholdStatePolicy.forActor(state, actorKey);
  const requested = HouseholdViewPolicy.fromSearch(location.search);
  const view = HouseholdViewPolicy.normalize(
    requested,
    visibleState.household?.members ?? [],
  );
  const navigationView = visibleState.household ? view : requested;
  const canonicalSearch = visibleState.household
    ? HouseholdViewPolicy.search(view, location.search)
    : location.search.slice(1);
  useEffect(() => {
    if (!visibleState.household || canonicalSearch === location.search.slice(1))
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
    visibleState.household,
  ]);
  return useMemo(
    () => ({
      ...visibleState,
      actor,
      view,
      reportView: HouseholdViewPolicy.report(view) as ReportView,
      select(next: HouseholdView) {
        if (!visibleState.household) return;
        const normalized = HouseholdViewPolicy.normalize(
          next,
          visibleState.household.members,
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
        return HouseholdViewPolicy.path(pathname, navigationView);
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
      navigationView,
      visibleState,
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

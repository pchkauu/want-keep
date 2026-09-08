import { useSyncExternalStore } from "react";
import { CashAccountForm } from "./CashAccountForm";
import { AccountList } from "./AccountList";
import type { CashController } from "./cash-controller";
import type { HouseholdMember, HouseholdView } from "@/features/household";

export function CashAccounts({
  controller,
  view,
  members,
}: {
  controller: CashController;
  view: HouseholdView;
  members: readonly HouseholdMember[];
}) {
  const created = useSyncExternalStore(
    controller.subscribe,
    () => controller.snapshot().confirmed?.id,
    () => undefined,
  );
  return (
    <>
      <CashAccountForm controller={controller} />
      <AccountList
        key={created}
        api={controller.api}
        view={view}
        members={members}
      />
    </>
  );
}

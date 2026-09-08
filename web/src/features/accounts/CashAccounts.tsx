import { useSyncExternalStore } from "react";
import { CashAccountForm } from "./CashAccountForm";
import { AccountList } from "./AccountList";
import type { CashController } from "./cash-controller";

export function CashAccounts({ controller }: { controller: CashController }) {
  const created = useSyncExternalStore(
    controller.subscribe,
    () => controller.snapshot().confirmed?.id,
    () => undefined,
  );
  return (
    <>
      <CashAccountForm controller={controller} />
      <AccountList key={created} api={controller.api} />
    </>
  );
}

import { useEffect, useSyncExternalStore, type ReactNode } from "react";
import { LocaleContext, type LocaleController } from "./locale";

export function LocaleProvider({
  controller,
  children,
}: {
  controller: LocaleController;
  children: ReactNode;
}) {
  const locale = useSyncExternalStore(
    controller.subscribe,
    controller.snapshot,
    controller.snapshot,
  );
  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);
  return (
    <LocaleContext.Provider value={controller}>
      {children}
    </LocaleContext.Provider>
  );
}

import { createContext, useContext, useSyncExternalStore } from "react";
import type { SessionController } from "./session-controller";

export const IdentityContext = createContext<SessionController | null>(null);
export function useIdentity() {
  const controller = useContext(IdentityContext);
  if (!controller) throw new Error("Identity provider is required");
  const state = useSyncExternalStore(
    controller.subscribe,
    controller.snapshot,
    controller.snapshot,
  );
  return { ...state, controller, api: controller.api };
}

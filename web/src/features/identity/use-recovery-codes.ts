import { useEffect, useRef, useState } from "react";
import { useIdentity } from "./context";

export function useRecoveryCodes() {
  const { controller } = useIdentity();
  const [codes, update] = useState<string[]>();
  const pending = useRef(false);
  const owner = useRef<string | undefined>(undefined);
  useEffect(
    () =>
      controller.subscribe(() => {
        const session = controller.snapshot();
        if (
          session.status !== "active" ||
          session.member?.userId !== owner.current
        ) {
          owner.current = undefined;
          pending.current = false;
          update(undefined);
        }
      }),
    [controller],
  );
  const setCodes = (value?: string[]) => {
    const session = controller.snapshot();
    owner.current =
      value && session.status === "active" ? session.member?.userId : undefined;
    pending.current = Boolean(owner.current && value);
    update(owner.current ? value : undefined);
  };
  return { codes, setCodes, hasUnsavedCodes: () => pending.current };
}

import { useEffect, useRef, useState } from "react";
import { ApiFailure } from "@/api/http";
import { WebAuthn } from "./webauthn";

export function useCeremony() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiFailure>();
  const lock = useRef(false);
  const live = useRef(true);
  const [webauthn] = useState(() => new WebAuthn());
  useEffect(() => {
    live.current = true;
    webauthn.activate();
    return () => {
      live.current = false;
      webauthn.close();
    };
  }, [webauthn]);
  const run = async (action: () => Promise<void>) => {
    if (lock.current) return;
    lock.current = true;
    setBusy(true);
    setError(undefined);
    try {
      await action();
    } catch (failure) {
      if (live.current)
        setError(
          failure instanceof ApiFailure
            ? failure
            : new ApiFailure("service_unavailable"),
        );
    } finally {
      lock.current = false;
      if (live.current) setBusy(false);
    }
  };
  return {
    busy,
    error,
    webauthn,
    run,
    clearError: () => setError(undefined),
    isLive: () => live.current,
  };
}

import { useEffect, type ReactNode } from "react";
import { Navigate, useLocation } from "react-router";
import { Button } from "@/design-system/components/button";
import { useIdentity } from "@/features/identity";
import { useLocale } from "@/locales/locale";
import { RouteContext } from "@/navigation/route-context";
import { useServices } from "./services";

export function SessionBoundary({ children }: { children: ReactNode }) {
  const { status, member, controller, logoutUnconfirmed } = useIdentity();
  const { t } = useLocale();
  const services = useServices();
  const location = useLocation();
  useEffect(() => {
    if (status !== "expired" || !member) return;
    const path = RouteContext.internal(location.pathname + location.search);
    if (path) services.rememberReturn(member.userId, path);
  }, [status, member, location.pathname, location.search, services]);
  if (status === "active") return children;
  if (status === "expired" || status === "anonymous")
    return <Navigate to="/login" replace />;
  return (
    <main className="access-wait bg-background">
      <img src="/brand/logo_512px.svg" width="40" height="45" alt="" />
      <p role="status">
        {t(
          logoutUnconfirmed
            ? "logoutUnconfirmed"
            : status === "unavailable"
              ? "service_unavailable"
              : "checking",
        )}
      </p>
      {status === "unavailable" && (
        <Button onClick={() => void controller.verify()}>{t("retry")}</Button>
      )}
    </main>
  );
}

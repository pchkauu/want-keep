import { useEffect } from "react";
import { Link, useNavigate } from "react-router";
import { ApiFailure } from "@/api/http";
import { Button } from "@/design-system/components/button";
import { useLocale } from "@/locales/locale";
import { AuthLayout } from "./AuthLayout";
import { AuthFeedback } from "./AuthFeedback";
import { useIdentity } from "./context";
import { useCeremony } from "./use-ceremony";

export function LoginPage({
  destination = "/overview",
}: {
  destination?: string;
}) {
  const { t } = useLocale();
  const { status, controller, api } = useIdentity();
  const action = useCeremony();
  const navigate = useNavigate();
  useEffect(() => {
    if (status === "active") void navigate(destination, { replace: true });
  }, [status, destination, navigate]);
  return (
    <AuthLayout login>
      <div className="login-action">
        <Button
          className="login-button"
          disabled={action.busy || status === "checking"}
          onClick={() =>
            void action.run(async () => {
              await controller.verify();
              if (!action.isLive() || controller.snapshot().status === "active")
                return;
              if (
                !["anonymous", "expired"].includes(controller.snapshot().status)
              )
                throw new ApiFailure("service_unavailable");
              const ticket = controller.ticket();
              const member = await api.login(action.webauthn, "login");
              if (!action.isLive()) return;
              controller.accept(member, ticket);
              controller.announce();
            })
          }
        >
          {t("login")}
        </Button>
        {status === "checking" && <p role="status">{t("checking")}</p>}
        {status === "expired" && (
          <p role="status">
            {t("expired")} {t("restoreSame")}
          </p>
        )}
        {status === "unavailable" && (
          <>
            <p role="status">{t("service_unavailable")}</p>
            <Button
              variant="secondary"
              onClick={() => void controller.verify()}
            >
              {t("retry")}
            </Button>
          </>
        )}
        <AuthFeedback busy={action.busy} error={action.error} enrollment />
        <Link className="access-link login-help" to="/recovery">
          {t("loginHelp")}
        </Link>
      </div>
    </AuthLayout>
  );
}

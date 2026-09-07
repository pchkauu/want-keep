import { Link } from "react-router";
import { ApiFailure } from "@/api/http";
import { useLocale } from "@/locales/locale";
import { ru, type MessageKey } from "@/locales/messages";
import { Button } from "@/design-system/components/button";
import { useIdentity } from "./context";

export function AuthFeedback({
  busy,
  error,
  enrollment = false,
}: {
  busy?: boolean;
  error?: ApiFailure;
  enrollment?: boolean;
}) {
  const { t } = useLocale();
  const { controller, status } = useIdentity();
  const unknown =
    error && ["network_unconfirmed", "invalid_response"].includes(error.code);
  const code =
    error?.code === "reauthentication_required"
      ? "fresh_authentication_required"
      : error?.code === "authentication_attempt_rejected"
        ? "attempt_invalid"
        : error?.code;
  const key =
    code && Object.hasOwn(ru, code)
      ? (code as MessageKey)
      : "service_unavailable";
  if (enrollment && unknown && status === "active")
    return (
      <div className="auth-feedback" role="status">
        <p>{t("accessRecovered")}</p>
        <Link className="access-link" to="/settings/security">
          {t("lostCodes")}
        </Link>
      </div>
    );
  return (
    <div className="auth-feedback" aria-live="polite" aria-atomic="true">
      {busy && <p role="status">{t("prompt")}</p>}
      {error && <p role="alert">{t(key)}</p>}
      {unknown && enrollment && (
        <>
          <p>{t("unknownEnrollment")}</p>
          <div className="access-actions">
            <Button
              variant="secondary"
              onClick={() => void controller.verify()}
            >
              {t("retry")}
            </Button>
            <Link className="access-link" to="/login">
              {t("login")}
            </Link>
            <Link className="access-link" to="/settings/security">
              {t("lostCodes")}
            </Link>
          </div>
        </>
      )}
    </div>
  );
}

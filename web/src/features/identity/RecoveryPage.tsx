import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { Button } from "@/design-system/components/button";
import { Field, FieldLabel } from "@/design-system/components/field";
import { Input } from "@/design-system/components/input";
import { LeaveGuard } from "@/navigation/LeaveGuard";
import { useLocale } from "@/locales/locale";
import { AuthLayout } from "./AuthLayout";
import { AuthFeedback } from "./AuthFeedback";
import { RecoveryCodes } from "./RecoveryCodes";
import { useRecoveryCodes } from "./use-recovery-codes";
import { useIdentity } from "./context";
import { useCeremony } from "./use-ceremony";

export function RecoveryPage() {
  const { t } = useLocale();
  const { controller, api, status } = useIdentity();
  const action = useCeremony();
  const navigate = useNavigate();
  const [code, setCode] = useState("");
  const [grant, setGrant] = useState<{ token: string; expiresAt: string }>();
  const { codes, setCodes, hasUnsavedCodes } = useRecoveryCodes();
  const unknown = Boolean(
    action.error &&
    ["network_unconfirmed", "invalid_response"].includes(action.error.code),
  );
  return (
    <AuthLayout title={t("recovery")} description={t("recoveryIntro")}>
      <LeaveGuard
        dirty={() =>
          hasUnsavedCodes() || (status !== "active" && Boolean(code || grant))
        }
      />
      {codes && status === "active" ? (
        <RecoveryCodes
          codes={codes}
          onDone={() => {
            setCodes(undefined);
            void navigate("/overview");
          }}
        />
      ) : status === "active" ? (
        <Link className="access-link" to="/settings/security">
          {t("lostCodes")}
        </Link>
      ) : grant ? (
        <section className="access-form">
          <p>{t("recoveryConsumed")}</p>
          <Button
            disabled={action.busy || unknown}
            onClick={() =>
              void action.run(async () => {
                const ticket = controller.ticket();
                const result = await api.recoverPasskey(
                  grant.token,
                  action.webauthn,
                  t("keyDefault"),
                );
                if (!action.isLive()) return;
                controller.accept(result.me, ticket);
                controller.announce();
                setGrant(undefined);
                setCodes(result.codes);
              })
            }
          >
            {t("newPasskey")}
          </Button>
          <Button
            variant="secondary"
            disabled={action.busy}
            onClick={() => {
              setGrant(undefined);
              action.clearError();
            }}
          >
            {t("cancel")}
          </Button>
        </section>
      ) : (
        <form
          className="access-form"
          onSubmit={(event) => {
            event.preventDefault();
            void action.run(async () => {
              const result = await api.recovery(code);
              if (action.isLive()) {
                setCode("");
                setGrant(result);
              }
            });
          }}
        >
          <Field>
            <FieldLabel>{t("recoveryCode")}</FieldLabel>
            <Input
              type="password"
              autoComplete="off"
              required
              maxLength={1024}
              value={code}
              onValueChange={setCode}
              disabled={action.busy}
            />
          </Field>
          <Button
            type="submit"
            disabled={action.busy || unknown || status === "checking"}
          >
            {t("checkCode")}
          </Button>
          {unknown && (
            <Button
              variant="secondary"
              onClick={() => {
                setCode("");
                action.clearError();
              }}
            >
              {t("cancel")}
            </Button>
          )}
        </form>
      )}
      <AuthFeedback busy={action.busy} error={action.error} enrollment />
    </AuthLayout>
  );
}

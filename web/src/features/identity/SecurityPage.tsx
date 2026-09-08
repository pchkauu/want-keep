import { LeaveGuard } from "@/navigation/LeaveGuard";
import { Button } from "@/design-system/components/button";
import { useLocale } from "@/locales/locale";
import { AuthFeedback } from "./AuthFeedback";
import { RecoveryCodes } from "./RecoveryCodes";
import { useRecoveryCodes } from "./use-recovery-codes";
import { useIdentity } from "./context";
import { useCeremony } from "./use-ceremony";

export function SecurityPage() {
  const { t } = useLocale();
  const { controller, api } = useIdentity();
  const action = useCeremony();
  const { codes, setCodes, hasUnsavedCodes } = useRecoveryCodes();
  return (
    <section className="access-panel">
      <LeaveGuard dirty={hasUnsavedCodes} />
      <h2>{t("security")}</h2>
      <p>{t("securityDescription")}</p>
      {codes ? (
        <RecoveryCodes codes={codes} onDone={() => setCodes(undefined)} />
      ) : (
        <Button
          disabled={action.busy}
          onClick={() =>
            void action.run(async () => {
              const ticket = controller.ticket();
              const member = await api.login(
                action.webauthn,
                "reauthentication",
              );
              if (!action.isLive()) return;
              controller.accept(member, ticket);
              const result = await api.replaceCodes();
              if (action.isLive()) setCodes(result);
            })
          }
        >
          {t("replaceCodes")}
        </Button>
      )}
      <AuthFeedback busy={action.busy} error={action.error} />
    </section>
  );
}

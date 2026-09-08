import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { Button } from "@/design-system/components/button";
import { Field, FieldLabel } from "@/design-system/components/field";
import { Input } from "@/design-system/components/input";
import { LeaveGuard } from "@/navigation/LeaveGuard";
import { useLocale } from "@/locales/locale";
import { AuthLayout } from "./AuthLayout";
import { AuthFeedback } from "./AuthFeedback";
import { ProfileFields } from "./ProfileFields";
import { RecoveryCodes } from "./RecoveryCodes";
import { useRecoveryCodes } from "./use-recovery-codes";
import { useIdentity } from "./context";
import { useCeremony } from "./use-ceremony";
import type { Asset } from "./model";

export function SetupPage() {
  const { t, locale } = useLocale();
  const { controller, api, status } = useIdentity();
  const action = useCeremony();
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [householdName, setHouseholdName] = useState("");
  const [token, setToken] = useState("");
  const [timezone, setTimezone] = useState(
    () => Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
  );
  const [asset, setAsset] = useState<Asset>("RUB");
  const { codes, setCodes, hasUnsavedCodes } = useRecoveryCodes();
  const unknown =
    action.error &&
    ["network_unconfirmed", "invalid_response"].includes(action.error.code);
  return (
    <AuthLayout title={t("setup")} description={t("setupDescription")}>
      <LeaveGuard
        dirty={() =>
          hasUnsavedCodes() || (status !== "active" && Boolean(name || token))
        }
      />
      {codes && status === "active" ? (
        <RecoveryCodes
          codes={codes}
          onDone={() => {
            setCodes(undefined);
            void navigate("/onboarding");
          }}
        />
      ) : status === "active" ? (
        <Link className="access-link" to="/onboarding">
          {t("toOnboarding")}
        </Link>
      ) : (
        <form
          className="access-form"
          onSubmit={(event) => {
            event.preventDefault();
            void action.run(async () => {
              const ticket = controller.ticket();
              const result = await api.setup(
                {
                  name,
                  householdName,
                  timezone,
                  locale,
                  reportingAsset: asset,
                  bootstrapToken: token,
                },
                action.webauthn,
                t("keyDefault"),
              );
              if (!action.isLive()) return;
              controller.accept(result.me, ticket);
              controller.announce();
              setToken("");
              setCodes(result.codes);
            });
          }}
        >
          <Field>
            <FieldLabel>{t("bootstrapToken")}</FieldLabel>
            <Input
              type="password"
              autoComplete="off"
              value={token}
              onValueChange={setToken}
              required
              maxLength={1024}
              disabled={action.busy}
            />
          </Field>
          <ProfileFields
            name={name}
            asset={asset}
            onName={setName}
            onAsset={setAsset}
            disabled={action.busy}
          />
          <Field>
            <FieldLabel>{t("householdName")}</FieldLabel>
            <Input
              value={householdName}
              onValueChange={setHouseholdName}
              required
              maxLength={2000}
              disabled={action.busy}
            />
          </Field>
          <Field>
            <FieldLabel>{t("timezone")}</FieldLabel>
            <Input
              value={timezone}
              onValueChange={setTimezone}
              required
              maxLength={100}
              disabled={action.busy}
              placeholder="Europe/Moscow"
            />
          </Field>
          <Button
            type="submit"
            disabled={action.busy || status === "checking" || Boolean(unknown)}
          >
            {t("createFamily")}
          </Button>
        </form>
      )}
      <AuthFeedback busy={action.busy} error={action.error} enrollment />
    </AuthLayout>
  );
}

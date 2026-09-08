import { useEffect, useState } from "react";
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
import type { Asset, InvitationPreview } from "./model";

export function InvitePage({
  initialToken,
  consumed,
}: {
  initialToken: string;
  consumed: () => void;
}) {
  const { t, locale } = useLocale();
  const { controller, api, status } = useIdentity();
  const action = useCeremony();
  const navigate = useNavigate();
  const [token, setToken] = useState(() => initialToken);
  useEffect(() => consumed(), [consumed]);
  const [name, setName] = useState("");
  const [asset, setAsset] = useState<Asset>("RUB");
  const [preview, setPreview] = useState<InvitationPreview>();
  const { codes, setCodes, hasUnsavedCodes } = useRecoveryCodes();
  const unknown =
    action.error &&
    ["network_unconfirmed", "invalid_response"].includes(action.error.code);
  return (
    <AuthLayout title={t("invite")} description={t("invitationPrivacy")}>
      <LeaveGuard
        dirty={() =>
          hasUnsavedCodes() || (status !== "active" && Boolean(token || name))
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
        <>
          <p>{t(unknown ? "accessRecovered" : "already_authenticated")}</p>
          <Link
            className="access-link"
            to={unknown ? "/onboarding" : "/overview"}
          >
            {t(unknown ? "toOnboarding" : "toOverview")}
          </Link>
          {!unknown && (
            <Button
              variant="secondary"
              onClick={() => void action.run(() => controller.signOut())}
            >
              {t("logout")}
            </Button>
          )}
        </>
      ) : (
        <form
          className="access-form"
          onSubmit={(event) => {
            event.preventDefault();
            consumed();
            void action.run(async () => {
              if (!preview) {
                const result = await api.preview(token);
                if (action.isLive()) setPreview(result);
                return;
              }
              const ticket = controller.ticket();
              const result = await api.accept(
                token,
                { name, locale, reportingAsset: asset },
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
          {!preview ? (
            <Field>
              <FieldLabel>{t("invitationToken")}</FieldLabel>
              <Input
                type="password"
                autoComplete="off"
                required
                maxLength={1024}
                value={token}
                onValueChange={setToken}
                disabled={action.busy}
              />
            </Field>
          ) : (
            <>
              <div className="invitation-preview">
                <h2>{preview.householdName}</h2>
                <p>
                  {t("invitedBy")}: {preview.inviterName}
                </p>
              </div>
              <ProfileFields
                name={name}
                asset={asset}
                onName={setName}
                onAsset={setAsset}
                disabled={action.busy}
              />
            </>
          )}
          <Button
            type="submit"
            disabled={action.busy || status === "checking" || Boolean(unknown)}
          >
            {t(preview ? "acceptInvitation" : "checkInvitation")}
          </Button>
          {preview && (
            <Button
              variant="secondary"
              disabled={action.busy || Boolean(unknown)}
              onClick={() => setPreview(undefined)}
            >
              {t("back")}
            </Button>
          )}
        </form>
      )}
      <AuthFeedback busy={action.busy} error={action.error} enrollment />
    </AuthLayout>
  );
}

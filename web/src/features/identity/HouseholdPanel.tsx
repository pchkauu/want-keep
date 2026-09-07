import { useEffect, useState } from "react";
import { Button } from "@/design-system/components/button";
import { Input } from "@/design-system/components/input";
import { useLocale } from "@/locales/locale";
import { AuthFeedback } from "./AuthFeedback";
import { useIdentity } from "./context";
import { useCeremony } from "./use-ceremony";
import type { Household, Invitation } from "./model";

export function HouseholdPanel() {
  const { t, locale } = useLocale();
  const { api, controller } = useIdentity();
  const action = useCeremony();
  const [data, setData] = useState<{
    family: Household;
    invitation: Invitation;
  }>();
  const [link, setLink] = useState("");
  const [copyState, setCopyState] = useState<"copied" | "copyFailed">();
  const [refresh, setRefresh] = useState(0);
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 30_000);
    return () => clearInterval(timer);
  }, []);
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    let live = true;
    void Promise.all([api.household(), api.invitations()]).then(
      ([family, invitation]) => {
        if (live) {
          setData({ family, invitation });
          setFailed(false);
        }
      },
      () => {
        if (live) setFailed(true);
      },
    );
    return () => {
      live = false;
    };
  }, [api, refresh]);
  const current = data?.invitation.current;
  const status =
    current?.status === "accepted"
      ? "invitationAccepted"
      : current?.status === "revoked"
        ? "invitationRevoked"
        : current
          ? Date.parse(current.expiresAt) <= now
            ? "invitationExpired"
            : "invitationActive"
          : "noInvitation";
  return (
    <section className="access-panel">
      <h2>{t("family")}</h2>
      {data ? (
        <>
          <h3>{data.family.name}</h3>
          <ul className="access-members">
            {data.family.members.map((member) => (
              <li key={member.id}>
                <span className="member-avatar" aria-hidden="true">
                  {member.name.slice(0, 1)}
                </span>
                {member.name}
              </li>
            ))}
          </ul>
          <p>{t(status)}</p>
          {current && (
            <p>
              {t("expiresAt")}:{" "}
              {new Intl.DateTimeFormat(locale, {
                dateStyle: "medium",
                timeStyle: "short",
              }).format(new Date(current.expiresAt))}
            </p>
          )}
          {data.family.members.length < data.family.maximum && (
            <Button
              disabled={action.busy || failed || Boolean(action.error)}
              onClick={() =>
                void action.run(async () => {
                  setLink("");
                  setCopyState(undefined);
                  const ticket = controller.ticket();
                  const member = await api.login(
                    action.webauthn,
                    "reauthentication",
                  );
                  if (!action.isLive()) return;
                  controller.accept(member, ticket);
                  const result = await api.issue(data.invitation.revision);
                  if (action.isLive()) {
                    setData({ ...data, invitation: result.state });
                    setLink(`${window.location.origin}/invite#${result.token}`);
                  }
                })
              }
            >
              {t(current ? "reissueInvitation" : "issueInvitation")}
            </Button>
          )}
          {current?.status === "active" && (
            <Button
              variant="secondary"
              disabled={action.busy || failed || Boolean(action.error)}
              onClick={() =>
                void action.run(async () => {
                  const invitation = await api.revoke(
                    current.id,
                    data.invitation.revision,
                  );
                  if (action.isLive()) {
                    setData({ ...data, invitation });
                    setLink("");
                  }
                })
              }
            >
              {t("revokeInvitation")}
            </Button>
          )}
          {link && (
            <div className="access-notice">
              <p>{t("invitationOnce")}</p>
              <Input
                aria-label={t("invitationLink")}
                value={link}
                readOnly
                autoComplete="off"
              />
              <Button
                variant="secondary"
                onClick={() => {
                  void navigator.clipboard.writeText(link).then(
                    () => setCopyState("copied"),
                    () => setCopyState("copyFailed"),
                  );
                }}
              >
                {t("copyInvitation")}
              </Button>
              {copyState && <p role="status">{t(copyState)}</p>}
            </div>
          )}
        </>
      ) : (
        <p role="status">{t(failed ? "service_unavailable" : "pending")}</p>
      )}
      <AuthFeedback busy={action.busy} error={action.error} />
      {(failed || action.error) && (
        <div className="access-notice">
          <p>{t("invitationLost")}</p>
          <Button
            disabled={action.busy}
            variant="secondary"
            onClick={() => {
              setLink("");
              setFailed(true);
              action.clearError();
              setRefresh((value) => value + 1);
            }}
          >
            {t("retry")}
          </Button>
        </div>
      )}
    </section>
  );
}

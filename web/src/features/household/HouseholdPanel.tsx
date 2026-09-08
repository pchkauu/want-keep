import { useEffect, useState } from "react";
import { Link } from "react-router";
import { Badge } from "@/design-system/components/badge";
import { Button } from "@/design-system/components/button";
import { Input } from "@/design-system/components/input";
import { useIdentity, useOwnPasskeyConfirmation } from "@/features/identity";
import { useLocale } from "@/locales/locale";
import { ru, type MessageKey } from "@/locales/messages";
import { useHousehold } from "./use-household";

export function HouseholdPanel({ compact = false }: { compact?: boolean }) {
  const { t, locale } = useLocale();
  const identity = useIdentity();
  const {
    status,
    refreshing,
    actor,
    household,
    invitation,
    invitationLoadStatus,
    invitationError,
    error,
    controller,
    refresh,
    path,
  } = useHousehold();
  const action = useOwnPasskeyConfirmation();
  const [link, setLink] = useState("");
  const [copyState, setCopyState] = useState<"copied" | "copyFailed">();
  const [now, setNow] = useState(Date.now);
  const [offline, setOffline] = useState(() => !navigator.onLine);

  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 30_000);
    const updateNetwork = () => setOffline(!navigator.onLine);
    window.addEventListener("online", updateNetwork);
    window.addEventListener("offline", updateNetwork);
    return () => {
      clearInterval(timer);
      window.removeEventListener("online", updateNetwork);
      window.removeEventListener("offline", updateNetwork);
    };
  }, []);

  const current = invitation?.current;
  const invitationStateLabel: MessageKey =
    current?.status === "accepted"
      ? "invitationAccepted"
      : current?.status === "revoked"
        ? "invitationRevoked"
        : current
          ? Date.parse(current.expiresAt) <= now
            ? "invitationExpired"
            : "invitationActive"
          : "noInvitation";
  const actionCode = action.error?.code;
  const actionMessage: MessageKey =
    actionCode && Object.hasOwn(ru, actionCode)
      ? (actionCode as MessageKey)
      : "service_unavailable";
  const unknown =
    actionCode === "network_unconfirmed" || actionCode === "invalid_response";
  const conflict = actionCode === "version_conflict";
  const activeCount =
    household?.members.filter((member) => member.status === "active").length ??
    0;

  return (
    <section
      className="access-panel household-panel"
      aria-busy={
        status === "loading" || refreshing || invitationLoadStatus === "loading"
      }
    >
      <div className="household-panel__heading">
        <div>
          <p className="start-hero__eyebrow">HOUSEHOLD SPACE</p>
          <h2>{t("family")}</h2>
        </div>
        {household && (
          <Badge>
            {activeCount}/{household.maximum} {t("memberSlotsUsed")}
          </Badge>
        )}
      </div>
      {status === "loading" || status === "idle" ? (
        <p role="status">{t("pending")}</p>
      ) : status === "error" || !household ? (
        <div className="access-notice">
          <p role="alert">
            {t(
              error?.code && Object.hasOwn(ru, error.code)
                ? (error.code as MessageKey)
                : "service_unavailable",
            )}
          </p>
          <Button variant="secondary" onClick={refresh}>
            {t("refreshHousehold")}
          </Button>
        </div>
      ) : (
        <>
          {refreshing && <p role="status">{t("refreshingHousehold")}</p>}
          <h3>{household.name}</h3>
          <p className="household-timezone">
            {t("familyTimezone")}: {household.timezone}
          </p>
          <ul className="access-members">
            {household.members.map((member) => (
              <li key={member.membershipId}>
                <span className="member-avatar" aria-hidden="true">
                  {member.name.slice(0, 1)}
                </span>
                <span>{member.name}</span>
                <Badge
                  tone={member.status === "active" ? "success" : "warning"}
                >
                  {t(
                    member.status === "active"
                      ? "membershipActive"
                      : "membershipPending",
                  )}
                </Badge>
              </li>
            ))}
          </ul>
          {invitationLoadStatus === "loading" ? (
            <p role="status">{t("pending")}</p>
          ) : invitationLoadStatus === "error" || invitationError ? (
            <div className="access-notice">
              <p role="alert">
                {t(
                  invitationError?.code &&
                    Object.hasOwn(ru, invitationError.code)
                    ? (invitationError.code as MessageKey)
                    : "service_unavailable",
                )}
              </p>
              <Button
                variant="secondary"
                disabled={refreshing}
                onClick={refresh}
              >
                {t("refreshHousehold")}
              </Button>
            </div>
          ) : invitation ? (
            <>
              <div className="household-invitation-state">
                <p>{t(invitationStateLabel)}</p>
                {current && (
                  <p>
                    {t("expiresAt")}:{" "}
                    {new Intl.DateTimeFormat(locale, {
                      dateStyle: "medium",
                      timeStyle: "short",
                    }).format(new Date(current.expiresAt))}
                  </p>
                )}
              </div>
              {activeCount < household.maximum ? (
                <Button
                  disabled={
                    action.busy ||
                    offline ||
                    refreshing ||
                    Boolean(error) ||
                    Boolean(action.error)
                  }
                  onClick={() =>
                    void action.run(async () => {
                      setLink("");
                      setCopyState(undefined);
                      const ticket = identity.controller.ticket();
                      const member = await identity.api.login(
                        action.webauthn,
                        "reauthentication",
                      );
                      if (!action.isLive()) return;
                      identity.controller.accept(member, ticket);
                      const result = await controller.issueInvitation(member);
                      if (!action.isLive()) return;
                      setLink(
                        `${window.location.origin}/invite#${result.token}`,
                      );
                    })
                  }
                >
                  {t(current ? "reissueInvitation" : "issueInvitation")}
                </Button>
              ) : (
                <p>{t("householdMemberLimitReached")}</p>
              )}
              {current?.status === "active" && (
                <Button
                  variant="secondary"
                  disabled={
                    action.busy ||
                    offline ||
                    refreshing ||
                    Boolean(error) ||
                    Boolean(action.error)
                  }
                  onClick={() =>
                    void action.run(async () => {
                      setLink("");
                      setCopyState(undefined);
                      await controller.revokeInvitation(actor);
                      if (!action.isLive()) return;
                    })
                  }
                >
                  {t("revokeInvitation")}
                </Button>
              )}
            </>
          ) : null}
          {offline && <p role="status">{t("householdOffline")}</p>}
          {error && (
            <div className="access-notice">
              <p role="alert">
                {t(
                  Object.hasOwn(ru, error.code)
                    ? (error.code as MessageKey)
                    : "service_unavailable",
                )}
              </p>
              <Button
                variant="secondary"
                disabled={refreshing}
                onClick={refresh}
              >
                {t("refreshHousehold")}
              </Button>
            </div>
          )}
          {action.busy && <p role="status">{t("prompt")}</p>}
          {action.error && (
            <div className="access-notice">
              <p role="alert">{t(actionMessage)}</p>
              {unknown && <p>{t("invitationUnknown")}</p>}
              {conflict && <p>{t("invitationConflict")}</p>}
              <Button
                variant="secondary"
                disabled={action.busy}
                onClick={() => {
                  setLink("");
                  action.clearError();
                  refresh();
                }}
              >
                {t("refreshHousehold")}
              </Button>
            </div>
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
          {compact && (
            <Link className="access-link" to={path("/settings/household")}>
              {t("manageHousehold")}
            </Link>
          )}
        </>
      )}
    </section>
  );
}

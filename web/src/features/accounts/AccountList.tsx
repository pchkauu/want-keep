import { useEffect, useRef, useState } from "react";
import { Button } from "@/design-system/components/button";
import { Badge } from "@/design-system/components/badge";
import type { HouseholdMember, HouseholdView } from "@/features/household";
import { useLocale } from "@/locales/locale";
import {
  AccountCollection,
  CashAccount,
  type AccountSummary,
} from "./cash-account";
import type { AccountsApi } from "./accounts-api";

export function AccountList({
  api,
  view,
  members,
}: {
  api: AccountsApi;
  view: HouseholdView;
  members: readonly HouseholdMember[];
}) {
  const { t, locale } = useLocale();
  const [data, setData] = useState<{
    items: AccountSummary[];
    cursor?: string;
  }>();
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const epoch = useRef(0);
  const loading = useRef(false);
  const [busy, setBusy] = useState(true);
  useEffect(() => {
    const ticket = ++epoch.current;
    loading.current = true;
    void api
      .list()
      .then(
        (value) => {
          if (ticket === epoch.current) {
            setData(value);
            setFailed(false);
          }
        },
        () => {
          if (ticket === epoch.current) setFailed(true);
        },
      )
      .finally(() => {
        if (ticket === epoch.current) {
          loading.current = false;
          setBusy(false);
        }
      });
    return () => {
      epoch.current = ticket + 1;
    };
  }, [api, attempt]);
  const groups = data
    ? AccountCollection.groups(data.items, view, members)
    : [];
  const visibleCount = groups.reduce(
    (count, group) => count + group.items.length,
    0,
  );
  const memberName = (id: string) =>
    members.find((member) => member.userId === id)?.name ?? t("unknownMember");
  return (
    <section className="access-panel">
      <h2>{t("currentAccounts")}</h2>
      {!data && !failed && <p role="status">{t("pending")}</p>}
      {failed && <p role="alert">{t("service_unavailable")}</p>}
      {data?.items.length === 0 && <p>{t("noAccounts")}</p>}
      {data && data.items.length > 0 && visibleCount === 0 && (
        <p>{t("noAccountsInView")}</p>
      )}
      <div className="access-account-groups">
        {groups.map((group) => (
          <section key={group.key} className="access-account-group">
            <h3>
              {group.scope === "household"
                ? t("householdAccounts")
                : `${t("personalAccounts")} · ${group.owner?.name ?? t("unknownMember")}`}
            </h3>
            <ul className="access-account-list">
              {group.items.map((account) => (
                <li key={account.id}>
                  <div className="account-heading">
                    <span>{account.name}</span>
                    <Badge>
                      {account.ownership.scope === "household"
                        ? t("householdOwnership")
                        : `${t("personalOwnership")} · ${memberName(account.ownership.personalOwnerId)}`}
                    </Badge>
                  </div>
                  <span className="access-amount">
                    {account.amount !== undefined
                      ? CashAccount.display(account.amount, locale)
                      : t("unknownAmount")}{" "}
                    {account.asset}
                  </span>
                  {account.externalAccountOwnerId && (
                    <small>
                      {t("platformOwner")}:{" "}
                      {memberName(account.externalAccountOwnerId)}
                    </small>
                  )}
                  {account.partial && <small>{t("partial")}</small>}
                  {account.stale && <small>{t("stale")}</small>}
                </li>
              ))}
            </ul>
          </section>
        ))}
      </div>
      {view.view === "member" && data && visibleCount < data.items.length && (
        <p className="account-filter-note">{t("accountFilterExplanation")}</p>
      )}
      {data?.cursor && (
        <Button
          variant="secondary"
          disabled={busy}
          onClick={() => {
            if (loading.current) return;
            const ticket = epoch.current;
            loading.current = true;
            setBusy(true);
            void api
              .list(data.cursor)
              .then(
                (page) => {
                  if (ticket === epoch.current) {
                    setData({
                      items: [...data.items, ...page.items],
                      cursor: page.cursor,
                    });
                    setFailed(false);
                  }
                },
                () => {
                  if (ticket === epoch.current) setFailed(true);
                },
              )
              .finally(() => {
                if (ticket === epoch.current) {
                  loading.current = false;
                  setBusy(false);
                }
              });
          }}
        >
          {t("next")}
        </Button>
      )}
      <Button
        variant="ghost"
        disabled={busy}
        onClick={() => {
          setBusy(true);
          setAttempt((value) => value + 1);
        }}
      >
        {t("retry")}
      </Button>
    </section>
  );
}

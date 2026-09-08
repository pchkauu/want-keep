import { useEffect, useRef, useState } from "react";
import { Button } from "@/design-system/components/button";
import { useLocale } from "@/locales/locale";
import { CashAccount, type AccountSummary } from "./cash-account";
import type { AccountsApi } from "./accounts-api";

export function AccountList({ api }: { api: AccountsApi }) {
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
  return (
    <section className="access-panel">
      <h2>{t("currentAccounts")}</h2>
      {!data && !failed && <p role="status">{t("pending")}</p>}
      {failed && <p role="alert">{t("service_unavailable")}</p>}
      {data?.items.length === 0 && <p>{t("noAccounts")}</p>}
      <ul className="access-account-list">
        {data?.items.map((account) => (
          <li key={account.id}>
            <span>{account.name}</span>
            <span className="access-amount">
              {account.amount !== undefined
                ? CashAccount.display(account.amount, locale)
                : t("unknownAmount")}{" "}
              {account.asset}
            </span>
            {account.partial && <small>{t("partial")}</small>}
            {account.stale && <small>{t("stale")}</small>}
          </li>
        ))}
      </ul>
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

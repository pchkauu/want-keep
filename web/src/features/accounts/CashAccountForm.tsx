import { useEffect, useSyncExternalStore } from "react";
import { Button } from "@/design-system/components/button";
import { DateField } from "@/design-system/components/date-field";
import { CalendarDate } from "@/design-system/components/calendar-date";
import { Field, FieldLabel } from "@/design-system/components/field";
import { Input } from "@/design-system/components/input";
import { MoneyField } from "@/design-system/components/money-field";
import { Select } from "@/design-system/components/select";
import { LeaveGuard } from "@/navigation/LeaveGuard";
import { useLocale } from "@/locales/locale";
import { ru, type MessageKey } from "@/locales/messages";
import { CashAccount } from "./cash-account";
import type { CashController } from "./cash-controller";

export function CashAccountForm({
  controller,
}: {
  controller: CashController;
}) {
  const { t, locale } = useLocale();
  const state = useSyncExternalStore(
    controller.subscribe,
    controller.snapshot,
    controller.snapshot,
  );
  const { draft, creation } = state;
  const locked = state.busy || Boolean(creation);
  useEffect(() => {
    void controller.loadRecent();
  }, [controller]);
  return (
    <section className="access-panel">
      <h2>{t("cashTitle")}</h2>
      <p>{t("cashDescription")}</p>
      <LeaveGuard
        dirty={!state.confirmed && Boolean(draft.name || draft.amount)}
      />
      {state.recent.length > 0 && (
        <section className="access-notice">
          <h3>{t("recentCommands")}</h3>
          {state.recent.map((command, index) => (
            <div className="access-actions" key={command.id}>
              <span>
                {t("commandPending")} {index + 1}
              </span>
              <Button
                variant="secondary"
                disabled={state.busy}
                onClick={() => void controller.check(command.id)}
              >
                {t("retry")}
              </Button>
              <Button
                variant="secondary"
                disabled={locked}
                onClick={() => controller.restore(command.id)}
              >
                {t("restoreRequest")}
              </Button>
            </div>
          ))}
        </section>
      )}
      {state.recoveryId && <p role="status">{t("restoreRequestHint")}</p>}
      <form
        className="access-form"
        onSubmit={(event) => {
          event.preventDefault();
          void controller.submit();
        }}
      >
        <Field>
          <FieldLabel>{t("accountName")}</FieldLabel>
          <Input
            required
            value={draft.name}
            maxLength={2000}
            disabled={locked}
            onValueChange={(name) => controller.draft({ name })}
          />
        </Field>
        <div className="access-form__row">
          <Field>
            <FieldLabel>{t("asset")}</FieldLabel>
            <Select
              label={t("asset")}
              value={draft.asset}
              placeholder={t("asset")}
              disabled={locked}
              options={["RUB", "USD", "USDT", "USDC", "BTC", "ETH"].map(
                (value) => ({ value, label: value }),
              )}
              onValueChange={(asset) => {
                if (asset) controller.draft({ asset });
              }}
            />
          </Field>
          <Field>
            <FieldLabel>{t("ownership")}</FieldLabel>
            <Select
              label={t("ownership")}
              value={draft.ownership}
              placeholder={t("ownership")}
              disabled={locked}
              options={[
                { value: "personal", label: t("personal") },
                { value: "household", label: t("shared") },
              ]}
              onValueChange={(ownership) => {
                if (ownership === "personal" || ownership === "household")
                  controller.draft({ ownership });
              }}
            />
          </Field>
        </div>
        <DateField
          label={t("openingDate")}
          value={draft.date}
          draft={
            draft.date
              ? CalendarDate.display(draft.date, locale)
              : draft.dateDraft
          }
          locale={locale}
          disabled={locked}
          onChange={(change) =>
            controller.draft({
              date: change.value ?? null,
              dateDraft: change.draft,
            })
          }
        />
        <Field>
          <FieldLabel>{t("openingBalance")}</FieldLabel>
          <MoneyField
            required
            value={draft.amount}
            assetLabel={draft.asset}
            disabled={locked}
            onValueChange={(amount) => controller.draft({ amount })}
          />
          <p className="wk-description">{t("exactAmount")}</p>
        </Field>
        {!creation && (
          <Button
            type="submit"
            disabled={
              locked ||
              !state.checkedRecent ||
              (state.recent.length > 0 && !state.recoveryId)
            }
          >
            {t(state.recoveryId ? "retrySame" : "createAccount")}
          </Button>
        )}
      </form>
      {state.error && (
        <p role="alert">
          {t(
            Object.hasOwn(ru, state.error.code)
              ? (state.error.code as MessageKey)
              : "service_unavailable",
          )}
        </p>
      )}
      {!state.checkedRecent && (
        <Button
          variant="secondary"
          disabled={state.busy}
          onClick={() => void controller.loadRecent()}
        >
          {t("retry")}
        </Button>
      )}
      {creation && !state.confirmed && (
        <div className="access-notice">
          <p role="status">
            {t(
              creation.state === "failed" ? "commandFailed" : "commandUnknown",
            )}
          </p>
          <div className="access-actions">
            {creation.state === "failed" ? (
              <Button onClick={() => controller.reset()}>{t("retry")}</Button>
            ) : (
              <Button
                disabled={state.busy}
                onClick={() => void controller.check(creation.id)}
              >
                {t("retry")}
              </Button>
            )}
            {state.retryOriginal && (
              <Button
                variant="secondary"
                disabled={state.busy}
                onClick={() => void controller.retry()}
              >
                {t("retrySame")}
              </Button>
            )}
            {creation.state === "pending" &&
              !state.retryOriginal &&
              state.recent.some((x) => x.id === creation.id) && (
                <Button
                  variant="secondary"
                  disabled={state.busy}
                  onClick={() => controller.restore(creation.id)}
                >
                  {t("restoreRequest")}
                </Button>
              )}
          </div>
        </div>
      )}
      {state.confirmed && (
        <div className="access-notice" role="status">
          <h3>
            {t("accountCreated")}: {state.confirmed.name}
          </h3>
          <p className="access-amount">
            {state.confirmed.amount !== undefined
              ? CashAccount.display(state.confirmed.amount, locale)
              : t("unknownAmount")}{" "}
            {state.confirmed.asset}
          </p>
          <Button variant="secondary" onClick={() => controller.reset()}>
            {t("newAccount")}
          </Button>
        </div>
      )}
    </section>
  );
}

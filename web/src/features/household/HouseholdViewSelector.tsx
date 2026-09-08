import {
  ToggleGroup,
  ToggleItem,
} from "@/design-system/components/toggle-group";
import { useLocale } from "@/locales/locale";
import { useHousehold } from "./use-household";
import { HouseholdViewPolicy } from "./model";

export function HouseholdViewSelector() {
  const { t } = useLocale();
  const { status, household, view, select } = useHousehold();
  const selected = HouseholdViewPolicy.key(view);
  return (
    <div className="household-view-selector">
      <span>{t("householdViewLabel")}</span>
      <ToggleGroup
        aria-label={t("householdViewLabel")}
        value={[selected]}
        onValueChange={(values) => {
          const next = values.at(-1);
          if (!next || next === selected) return;
          const parsed = HouseholdViewPolicy.fromKey(next);
          if (parsed) select(parsed);
        }}
      >
        <ToggleItem value="household" disabled={status !== "ready"}>
          {t("householdView")}
        </ToggleItem>
        {household?.members
          .filter((member) => member.status === "active")
          .map((member) => (
            <ToggleItem key={member.userId} value={`member:${member.userId}`}>
              {member.name}
            </ToggleItem>
          ))}
      </ToggleGroup>
    </div>
  );
}

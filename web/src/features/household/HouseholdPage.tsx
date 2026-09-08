import { Link } from "react-router";
import { HouseIcon, KeyRoundIcon, WalletCardsIcon } from "lucide-react";
import { useLocale } from "@/locales/locale";
import { useHousehold } from "./use-household";
import { HouseholdPanel } from "./HouseholdPanel";

export function HouseholdPage() {
  const { t } = useLocale();
  const { path } = useHousehold();
  return (
    <div className="household-page">
      <section className="start-hero household-hero">
        <div>
          <p className="start-hero__eyebrow">TWO PEOPLE. ONE CLEAR PICTURE.</p>
          <h2>{t("householdSettingsTitle")}</h2>
          <p>{t("householdSettingsDescription")}</p>
        </div>
        <HouseIcon aria-hidden="true" />
      </section>
      <div className="household-settings-grid">
        <HouseholdPanel />
        <div className="household-rules">
          <section className="access-panel">
            <WalletCardsIcon aria-hidden="true" />
            <h2>{t("householdResourcesTitle")}</h2>
            <p>{t("householdResourcesDescription")}</p>
          </section>
          <section className="access-panel">
            <KeyRoundIcon aria-hidden="true" />
            <h2>{t("personalAccessTitle")}</h2>
            <p>{t("personalAccessDescription")}</p>
            <Link className="access-link" to={path("/settings/security")}>
              {t("openSecurity")}
            </Link>
          </section>
          <section className="access-panel household-boundaries">
            <h2>{t("notAvailableInMvp")}</h2>
            <p>{t("householdBoundaries")}</p>
          </section>
        </div>
      </div>
    </div>
  );
}

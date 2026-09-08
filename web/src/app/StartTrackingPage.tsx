import { Link } from "react-router";
import { ArrowUpRightIcon } from "lucide-react";
import { AccountList, CashAccounts } from "@/features/accounts";
import { HouseholdPanel, useIdentity } from "@/features/identity";
import { useLocale } from "@/locales/locale";
import { useServices } from "./services";

export function StartTrackingPage({
  onboarding = false,
}: {
  onboarding?: boolean;
}) {
  const { t } = useLocale();
  const { member } = useIdentity();
  const services = useServices();
  if (!member) return null;
  return (
    <>
      <section className="start-hero">
        <div>
          <p className="start-hero__eyebrow">SAME GOALS. MORE FREEDOM.</p>
          <h2>{t(onboarding ? "onboarding" : "welcome")}</h2>
          <p>
            {t(onboarding ? "onboardingDescription" : "overviewDescription")}
          </p>
          <Link
            className="hero-link"
            to={onboarding ? "/overview" : "/onboarding"}
          >
            {t(onboarding ? "toOverview" : "toOnboarding")}
            <ArrowUpRightIcon aria-hidden="true" />
          </Link>
        </div>
        <img src="/brand/logo_512px.svg" width="455" height="512" alt="" />
      </section>
      {onboarding ? (
        <div className="start-grid">
          <div>
            <CashAccounts controller={services.cashAccount(member.userId)} />
          </div>
          <div>
            <HouseholdPanel />
          </div>
        </div>
      ) : (
        <div className="start-grid">
          <AccountList api={services.accounts} />
          <section className="access-panel">
            <h2>{t("invitePartner")}</h2>
            <p>{t("invitationPrivacy")}</p>
            <Link to="/onboarding" className="access-link">
              {t("toOnboarding")}
            </Link>
            <Link to="/settings/security" className="access-link">
              {t("lostCodes")}
            </Link>
          </section>
        </div>
      )}
    </>
  );
}

export function OnboardingPage() {
  return <StartTrackingPage onboarding />;
}

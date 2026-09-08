import { Link } from "react-router";
import { useHousehold } from "@/features/household";
import { useLocale } from "@/locales/locale";

export function UnavailablePage({ notFound = false }: { notFound?: boolean }) {
  const { t } = useLocale();
  const { path } = useHousehold();
  return (
    <section className="access-panel unavailable-page">
      <h2>{t(notFound ? "notFound" : "unavailable")}</h2>
      <p>{t("availableSteps")}</p>
      <Link className="access-link" to={path("/onboarding")}>
        {t("toOnboarding")}
      </Link>
    </section>
  );
}

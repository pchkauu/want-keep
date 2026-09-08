import { Link } from "react-router";
import { useLocale } from "@/locales/locale";

export function UnavailablePage({ notFound = false }: { notFound?: boolean }) {
  const { t } = useLocale();
  return (
    <section className="access-panel unavailable-page">
      <h2>{t(notFound ? "notFound" : "unavailable")}</h2>
      <p>{t("availableSteps")}</p>
      <Link className="access-link" to="/onboarding">
        {t("toOnboarding")}
      </Link>
    </section>
  );
}

import { useLocale } from "@/locales/locale";

export function RouteLoading() {
  const { t } = useLocale();
  return (
    <main className="access-wait bg-background">
      <img src="/brand/logo_512px.svg" width="40" height="45" alt="" />
      <p role="status">{t("checking")}</p>
    </main>
  );
}

import { Button } from "@/design-system/components/button";
import { useLocale } from "./locale";

export function LanguageSwitch() {
  const { locale, select, t } = useLocale();
  return (
    <div className="language-switch" role="group" aria-label={t("language")}>
      {(["ru", "en"] as const).map((value) => (
        <Button
          key={value}
          size="sm"
          variant="ghost"
          aria-pressed={locale === value}
          onClick={() => select(value)}
        >
          {value.toUpperCase()}
        </Button>
      ))}
    </div>
  );
}

import { useEffect, useState } from "react";
import { Button } from "../components/button";
import { Sidebar } from "../components/sidebar";
import { TooltipProvider } from "../components/tooltip";
import { Toaster } from "../components/toast";
import { CompositionSpecimen } from "./CompositionSpecimen";
import { ActionsSpecimen } from "./ActionsSpecimen";
import { FormSpecimen } from "./FormSpecimen";
import { NavigationSpecimen } from "./NavigationSpecimen";
import { TableSpecimen } from "./TableSpecimen";
import { OverlaySpecimen } from "./OverlaySpecimen";
import { StatesSpecimen } from "./StatesSpecimen";
import type { CatalogLocale } from "./catalog-locale";
import "./component-catalog.css";

const sections = [
  { id: "actions", ru: "Действия", en: "Actions" },
  { id: "forms", ru: "Формы", en: "Forms" },
  { id: "navigation", ru: "Навигация", en: "Navigation" },
  { id: "tables", ru: "Таблицы", en: "Tables" },
  { id: "overlays", ru: "Слои", en: "Overlays" },
  { id: "states", ru: "Состояния", en: "States" },
];

export function ComponentCatalog() {
  const [locale, setLocale] = useState<CatalogLocale>("ru");
  useEffect(() => {
    const previous = document.documentElement.lang;
    document.documentElement.lang = locale;
    return () => {
      document.documentElement.lang = previous;
    };
  }, [locale]);
  const [current, setCurrent] = useState(
    () => window.location.hash.slice(1) || "actions",
  );
  useEffect(() => {
    const onHash = () => setCurrent(window.location.hash.slice(1) || "actions");
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);
  const ru = locale === "ru";
  return (
    <TooltipProvider>
      <Toaster closeLabel={ru ? "Закрыть уведомление" : "Close notification"}>
        <div className="component-catalog" lang={locale}>
          <header className="catalog-header">
            <a
              href="/__design/tokens"
              className="catalog-brand"
              aria-label={ru ? "Want Keep — токены" : "Want Keep tokens"}
            >
              <img src="/brand/logo_512px.svg" alt="" />
              <span>Want Keep</span>
            </a>
            <p>
              {ru
                ? "Компоненты / только для разработки"
                : "Components / development only"}
            </p>
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setLocale(ru ? "en" : "ru")}
            >
              {ru ? "English" : "Русский"}
            </Button>
          </header>
          <main>
            <div className="catalog-intro">
              <p className="catalog-eyebrow">WANT KEEP / DESIGN SYSTEM</p>
              <h1>
                {ru ? "Финансы. Проще. Ближе." : "Finance. Clearer. Closer."}
              </h1>
              <p>
                {ru
                  ? "Интерактивные компоненты и синтетические примеры композиции. Никаких реальных финансовых действий."
                  : "Interactive components and synthetic compositions. No real financial actions."}
              </p>
            </div>
            <CompositionSpecimen locale={locale} />
            <div className="catalog-workspace">
              <aside>
                <Sidebar
                  label={ru ? "Разделы каталога" : "Catalog sections"}
                  items={sections.map((section) => ({
                    id: section.id,
                    label: section[locale],
                    href: `#${section.id}`,
                  }))}
                  currentId={current}
                  footer={
                    <a className="catalog-token-link" href="/__design/tokens">
                      {ru
                        ? "Токены и типографика ↗"
                        : "Tokens and typography ↗"}
                    </a>
                  }
                />
              </aside>
              <div className="catalog-sections">
                <section id="actions">
                  <ActionsSpecimen locale={locale} />
                </section>
                <section id="forms">
                  <FormSpecimen locale={locale} />
                </section>
                <section id="navigation">
                  <NavigationSpecimen locale={locale} />
                </section>
                <section id="tables">
                  <TableSpecimen locale={locale} />
                </section>
                <section id="overlays">
                  <OverlaySpecimen locale={locale} />
                </section>
                <section id="states">
                  <StatesSpecimen locale={locale} />
                </section>
              </div>
            </div>
          </main>
        </div>
      </Toaster>
    </TooltipProvider>
  );
}

import { Tabs, TabsList, TabsTrigger, TabsContent } from "../components/tabs";
import { Sidebar } from "../components/sidebar";
import { HomeIcon, WalletIcon, ChartNoAxesCombinedIcon } from "lucide-react";
import type { CatalogLocale } from "./catalog-locale";

export function NavigationSpecimen({ locale }: { locale: CatalogLocale }) {
  const ru = locale === "ru";
  return (
    <div className="catalog-stack">
      <div className="catalog-heading">
        <p className="catalog-eyebrow">
          03 / {ru ? "Навигация" : "Navigation"}
        </p>
        <h2>{ru ? "Контекст всегда рядом" : "Keep the context close"}</h2>
      </div>
      <Sidebar
        label={ru ? "Пример меню" : "Example menu"}
        currentId="overview"
        items={[
          {
            id: "overview",
            label: ru ? "Обзор" : "Overview",
            href: "#navigation",
            icon: <HomeIcon aria-hidden="true" />,
          },
          {
            id: "money",
            label: ru ? "Деньги" : "Money",
            href: "#tables",
            icon: <WalletIcon aria-hidden="true" />,
          },
          {
            id: "plan",
            label: ru ? "План" : "Plan",
            href: "#forms",
            icon: <ChartNoAxesCombinedIcon aria-hidden="true" />,
          },
        ]}
      />
      <Tabs defaultValue="family">
        <TabsList aria-label={ru ? "Представление" : "View"}>
          <TabsTrigger value="family">
            {ru ? "Семейное" : "Household"}
          </TabsTrigger>
          <TabsTrigger value="personal">
            {ru ? "Личное" : "Personal"}
          </TabsTrigger>
          <TabsTrigger value="unavailable" disabled>
            {ru ? "Недоступно" : "Unavailable"}
          </TabsTrigger>
        </TabsList>
        <TabsContent value="family">
          {ru
            ? "Совместные статьи и цели — отдельный общий блок."
            : "Shared items and goals have their own household block."}
        </TabsContent>
        <TabsContent value="personal">
          {ru
            ? "Выбор представления не меняет действующего пользователя."
            : "Changing the view does not change the acting user."}
        </TabsContent>
      </Tabs>
    </div>
  );
}

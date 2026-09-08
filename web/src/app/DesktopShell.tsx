import { useEffect, useRef, useState } from "react";
import { NavLink, Link, Outlet, useLocation } from "react-router";
import {
  ArrowUpRightIcon,
  BellIcon,
  ChartNoAxesCombinedIcon,
  HouseIcon,
  MenuIcon,
  MessageCircleIcon,
  PlugIcon,
  SettingsIcon,
  WalletIcon,
  CalendarDaysIcon,
} from "lucide-react";
import { Button } from "@/design-system/components/button";
import { useIdentity } from "@/features/identity";
import { LanguageSwitch } from "@/locales/LanguageSwitch";
import { useLocale } from "@/locales/locale";
import type { MessageKey } from "@/locales/messages";

const links = [
  { path: "/overview", key: "overview", icon: HouseIcon },
  { path: "/accounts", key: "accounts", icon: WalletIcon },
  { path: "/plan", key: "plan", icon: CalendarDaysIcon },
  { path: "/analytics", key: "analytics", icon: ChartNoAxesCombinedIcon },
  { path: "/chat", key: "chat", icon: MessageCircleIcon },
  { path: "/connections", key: "connections", icon: PlugIcon },
  { path: "/settings", key: "settings", icon: SettingsIcon },
] as const;

export function DesktopShell() {
  const { t } = useLocale();
  const { member, controller } = useIdentity();
  const location = useLocation();
  const [expanded, setExpanded] = useState(false);
  const [offline, setOffline] = useState(() => !navigator.onLine);
  const [logoutError, setLogoutError] = useState(false);
  const heading = useRef<HTMLHeadingElement>(null);
  const title: MessageKey =
    location.pathname === "/onboarding"
      ? "onboarding"
      : location.pathname === "/settings/security"
        ? "security"
        : location.pathname === "/notifications"
          ? "notifications"
          : (links.find((link) => link.path === location.pathname)?.key ??
            "notFound");
  useEffect(() => {
    heading.current?.focus();
  }, [location.pathname]);
  useEffect(() => {
    const update = () => setOffline(!navigator.onLine);
    window.addEventListener("online", update);
    window.addEventListener("offline", update);
    return () => {
      window.removeEventListener("online", update);
      window.removeEventListener("offline", update);
    };
  }, []);
  return (
    <div className="desktop-shell bg-background" data-expanded={expanded}>
      <a className="skip-link access-link" href="#main-content">
        {t("skip")}
      </a>
      <aside className="desktop-sidebar">
        <Link to="/overview" className="sidebar-brand" aria-label="Want Keep">
          <img src="/brand/logo_512px.svg" width="32" height="36" alt="" />
          <span>WANT KEEP</span>
        </Link>
        <Button
          className="sidebar-toggle"
          variant="ghost"
          size="icon"
          aria-label={t("menu")}
          aria-expanded={expanded}
          onClick={() => setExpanded((value) => !value)}
        >
          <MenuIcon aria-hidden="true" />
        </Button>
        <nav aria-label={t("navigation")} className="desktop-navigation">
          {links.map(({ path, key, icon: Icon }, index) => (
            <NavLink
              key={path}
              to={path}
              className={`desktop-nav-link${index === 5 ? " desktop-nav-link--lower" : ""}`}
              title={t(key)}
              onClick={() => setExpanded(false)}
            >
              <Icon aria-hidden="true" />
              <span>{t(key)}</span>
            </NavLink>
          ))}
        </nav>
        <p className="sidebar-caption">
          MONEY
          <br />
          UNDER CONTROL<span aria-hidden="true">✦</span>
        </p>
      </aside>
      <div className="desktop-workspace">
        <header className="desktop-header">
          <h1 tabIndex={-1} ref={heading}>
            {t(title)}
          </h1>
          <div className="desktop-header__actions">
            <LanguageSwitch />
            <Link
              className="icon-link"
              to="/notifications"
              aria-label={t("notifications")}
            >
              <BellIcon aria-hidden="true" />
            </Link>
            <div className="current-member">
              <small>{t("currentUser")}</small>
              <span>{member?.name}</span>
            </div>
            <Button
              variant="ghost"
              onClick={() => {
                void controller.signOut().catch(() => setLogoutError(true));
              }}
            >
              {t("logout")}
            </Button>
          </div>
        </header>
        {offline && (
          <p className="workspace-notice" role="status">
            {t("offline")}
          </p>
        )}
        {logoutError && <p role="alert">{t("network_unconfirmed")}</p>}
        <main id="main-content" className="desktop-content">
          <Outlet />
        </main>
        <footer className="desktop-footer">
          <span>WANT KEEP</span>
          <span>
            YOUR MONEY. MORE FREEDOM.{" "}
            <ArrowUpRightIcon size={14} aria-hidden="true" />
          </span>
        </footer>
      </div>
    </div>
  );
}

import type { ReactNode } from "react";
import { Link } from "react-router";
import { LanguageSwitch } from "@/locales/LanguageSwitch";
import { useLocale } from "@/locales/locale";

export function AuthLayout({
  title,
  description,
  children,
  login = false,
}: {
  title?: string;
  description?: string;
  children: ReactNode;
  login?: boolean;
}) {
  const { t } = useLocale();
  return (
    <main
      className={`access-auth bg-background ${login ? "access-auth--login" : ""}`}
    >
      <header className="access-auth__top">
        <Link to="/login" className="access-wordmark" aria-label="Want Keep">
          WANT <span>KEEP</span>
        </Link>
        <LanguageSwitch />
      </header>
      <div className="access-auth__body">
        {login ? (
          <div className="login-brand">
            <img src="/brand/logo_512px.svg" width="455" height="512" alt="" />
            <h1>Want Keep</h1>
          </div>
        ) : (
          <>
            <h1>{title}</h1>
            <p className="access-description">{description}</p>
          </>
        )}
        {children}
      </div>
      {!login && (
        <footer className="access-auth__footer">
          <Link className="access-link" to="/login">
            {t("back")}
          </Link>
        </footer>
      )}
    </main>
  );
}

import { createContext, useContext, useSyncExternalStore } from "react";
export type Locale = "ru" | "en";
import { en, ru, type MessageKey } from "./messages";

export class LocaleController {
  private language: Locale;
  private explicit = false;
  private listeners = new Set<() => void>();
  private readonly storage?: Pick<Storage, "getItem" | "setItem">;
  constructor(
    storage?: Pick<Storage, "getItem" | "setItem">,
    browserLanguage = "ru",
  ) {
    this.storage = storage;
    let saved: string | null = null;
    try {
      saved = storage?.getItem("want-keep.locale") ?? null;
    } catch {
      /* Browser storage may be denied. */
    }
    this.explicit = saved === "ru" || saved === "en";
    this.language = this.explicit
      ? (saved as Locale)
      : browserLanguage.toLowerCase().startsWith("en")
        ? "en"
        : "ru";
  }
  snapshot = () => this.language;
  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  };
  select(locale: Locale) {
    this.explicit = true;
    try {
      this.storage?.setItem("want-keep.locale", locale);
    } catch {
      /* In-memory selection still works. */
    }
    this.update(locale);
  }
  profile(locale: Locale) {
    if (!this.explicit) this.update(locale);
  }
  private update(locale: Locale) {
    this.language = locale;
    this.listeners.forEach((listener) => listener());
  }
}
export const LocaleContext = createContext<LocaleController | null>(null);
export function useLocale() {
  const controller = useContext(LocaleContext);
  if (!controller) throw new Error("Locale provider is required");
  const locale = useSyncExternalStore(
    controller.subscribe,
    controller.snapshot,
    controller.snapshot,
  );
  const t = (key: MessageKey): string => (locale === "ru" ? ru : en)[key];
  return { locale, t, select: (next: Locale) => controller.select(next) };
}

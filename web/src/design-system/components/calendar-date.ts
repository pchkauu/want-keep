import { format, isValid, parse } from "date-fns";

export type DateLocale = "ru" | "en";

export class CalendarDate {
  static readonly minimum = "0001-01-01";
  static readonly maximum = "9999-12-31";
  static pattern(locale: DateLocale) {
    return locale === "ru" ? "dd.MM.yyyy" : "MM/dd/yyyy";
  }

  static parse(text: string, locale: DateLocale): string | null | undefined {
    if (text === "") return null;
    const shape =
      locale === "ru" ? /^\d{2}\.\d{2}\.\d{4}$/ : /^\d{2}\/\d{2}\/\d{4}$/;
    if (!shape.test(text)) return undefined;
    const date = parse(text, this.pattern(locale), new Date(2000, 0, 1));
    if (!isValid(date) || format(date, this.pattern(locale)) !== text)
      return undefined;
    return format(date, "yyyy-MM-dd");
  }

  static toDate(value: string): Date;
  static toDate(value: string | null): Date | undefined;
  static toDate(value: string | null): Date | undefined {
    if (value === null) return undefined;
    if (!/^\d{4}-\d{2}-\d{2}$/.test(value))
      throw new TypeError("invalid_calendar_date");
    const date = parse(value, "yyyy-MM-dd", new Date(2000, 0, 1));
    if (!isValid(date) || format(date, "yyyy-MM-dd") !== value)
      throw new TypeError("invalid_calendar_date");
    return date;
  }

  static fromDate(date: Date): string {
    if (!isValid(date) || date.getFullYear() < 1 || date.getFullYear() > 9999)
      throw new TypeError("invalid_calendar_date");
    return format(date, "yyyy-MM-dd");
  }
  static display(value: string | null, locale: DateLocale): string {
    const date = this.toDate(value);
    return date ? format(date, this.pattern(locale)) : "";
  }
}

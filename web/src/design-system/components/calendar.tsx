import { DayPicker } from "@daypicker/react";
import { enUS, ru } from "@daypicker/react/locale";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronDownIcon,
} from "lucide-react";
import { CalendarDate, type DateLocale } from "./calendar-date";

export type CalendarProps = {
  locale: DateLocale;
  value: Date | undefined;
  onValueChange: (value: Date | undefined) => void;
};

export function Calendar({ locale, value, onValueChange }: CalendarProps) {
  const first = CalendarDate.toDate(CalendarDate.minimum);
  const last = CalendarDate.toDate(CalendarDate.maximum);
  return (
    <DayPicker
      mode="single"
      required
      locale={locale === "ru" ? ru : enUS}
      selected={value}
      startMonth={first}
      endMonth={last}
      disabled={[{ before: first }, { after: last }]}
      onSelect={onValueChange}
      weekStartsOn={1}
      autoFocus
      defaultMonth={value}
      className="wk-calendar"
      components={{
        Chevron: ({ orientation }) =>
          orientation === "left" ? (
            <ChevronLeftIcon aria-hidden="true" />
          ) : orientation === "right" ? (
            <ChevronRightIcon aria-hidden="true" />
          ) : (
            <ChevronDownIcon aria-hidden="true" />
          ),
      }}
    />
  );
}

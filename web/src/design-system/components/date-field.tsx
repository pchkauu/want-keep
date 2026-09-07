import { useId, useState } from "react";
import { CalendarIcon } from "lucide-react";
import { CalendarDate, type DateLocale } from "./calendar-date";
import { Calendar } from "./calendar";
import { Button } from "./button";
import { Input } from "./input";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverTitle,
} from "./popover";

export type DateFieldChange = {
  draft: string;
  value: string | null | undefined;
};
export type DateFieldProps = {
  id?: string;
  value: string | null;
  draft?: string;
  locale: DateLocale;
  label: string;
  disabled?: boolean;
  onChange: (change: DateFieldChange) => void;
};

export function DateField({
  id,
  value,
  draft,
  locale,
  label,
  disabled,
  onChange,
}: DateFieldProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const [open, setOpen] = useState(false);
  const [touched, setTouched] = useState(false);
  const text = draft ?? CalendarDate.display(value, locale);
  const parsed = CalendarDate.parse(text, locale);
  const invalid = touched && parsed === undefined;
  const hint = locale === "ru" ? "ДД.ММ.ГГГГ" : "MM/DD/YYYY";
  return (
    <div className="wk-field">
      <label className="wk-label" htmlFor={inputId}>
        {label}
      </label>
      <div className="wk-date-field">
        <Input
          id={inputId}
          disabled={disabled}
          value={text}
          aria-invalid={invalid}
          aria-describedby={`${inputId}-help`}
          onBlur={() => setTouched(true)}
          onValueChange={(next) =>
            onChange({ draft: next, value: CalendarDate.parse(next, locale) })
          }
        />
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger
            render={<Button variant="secondary" size="icon" />}
            disabled={disabled}
            aria-label={
              locale === "ru"
                ? `Открыть календарь: ${label}`
                : `Open calendar: ${label}`
            }
          >
            <CalendarIcon aria-hidden="true" />
          </PopoverTrigger>
          <PopoverContent>
            <PopoverTitle className="wk-label">{label}</PopoverTitle>
            <Calendar
              locale={locale}
              value={CalendarDate.toDate(parsed ?? null)}
              onValueChange={(date) => {
                const next = date ? CalendarDate.fromDate(date) : null;
                onChange({
                  value: next,
                  draft: CalendarDate.display(next, locale),
                });
                setTouched(false);
                setOpen(false);
              }}
            />
          </PopoverContent>
        </Popover>
      </div>
      <p
        id={`${inputId}-help`}
        className={invalid ? "wk-field-error" : "wk-description"}
      >
        {invalid
          ? locale === "ru"
            ? `Проверьте дату. Формат: ${hint}`
            : `Check the date. Format: ${hint}`
          : hint}
      </p>
    </div>
  );
}

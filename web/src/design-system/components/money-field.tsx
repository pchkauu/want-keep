import { Input, type InputProps } from "./input";

export type MoneyFieldProps = Omit<
  InputProps,
  | "value"
  | "defaultValue"
  | "type"
  | "inputMode"
  | "maxLength"
  | "onValueChange"
> & {
  value: string;
  assetLabel: string;
  onValueChange: (value: string) => void;
};

export function MoneyField({ assetLabel, ...props }: MoneyFieldProps) {
  return (
    <div className="wk-money-field">
      <Input
        {...props}
        type="text"
        inputMode="decimal"
        className="wk-amount-input"
      />
      <span className="wk-asset-label">{assetLabel}</span>
    </div>
  );
}

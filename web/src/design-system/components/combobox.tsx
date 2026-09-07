import { Combobox as Primitive } from "@base-ui/react/combobox";
import { CheckIcon } from "lucide-react";
import type { SelectOption } from "./select";

export type ComboboxProps = {
  id?: string;
  label: string;
  options: readonly SelectOption[];
  value: string | null;
  onValueChange: (value: string | null) => void;
  emptyLabel: string;
  disabled?: boolean;
};

export function Combobox({
  id,
  label,
  options,
  value,
  onValueChange,
  emptyLabel,
  disabled,
}: ComboboxProps) {
  return (
    <Primitive.Root
      items={options}
      value={options.find((option) => option.value === value) ?? null}
      onValueChange={(option) => onValueChange(option?.value ?? null)}
      itemToStringLabel={(option) => option.label}
      isItemEqualToValue={(item, selected) => item.value === selected.value}
      disabled={disabled}
    >
      <Primitive.Input id={id} aria-label={label} className="wk-input" />
      <Primitive.Portal>
        <Primitive.Positioner sideOffset={8} className="wk-positioner">
          <Primitive.Popup className="wk-select-popup">
            <Primitive.Empty className="wk-description">
              {emptyLabel}
            </Primitive.Empty>
            <Primitive.List>
              {(option: SelectOption) => (
                <Primitive.Item
                  key={option.value}
                  value={option}
                  disabled={option.disabled}
                  className="wk-select-item"
                >
                  {option.label}
                  <Primitive.ItemIndicator>
                    <CheckIcon aria-hidden="true" />
                  </Primitive.ItemIndicator>
                </Primitive.Item>
              )}
            </Primitive.List>
          </Primitive.Popup>
        </Primitive.Positioner>
      </Primitive.Portal>
    </Primitive.Root>
  );
}

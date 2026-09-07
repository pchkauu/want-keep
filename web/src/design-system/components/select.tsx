import { Select as Primitive } from "@base-ui/react/select";
import { CheckIcon, ChevronDownIcon } from "lucide-react";

export type SelectOption = { value: string; label: string; disabled?: boolean };
export type SelectProps = {
  id?: string;
  label: string;
  value: string | null;
  options: readonly SelectOption[];
  placeholder: string;
  disabled?: boolean;
  invalid?: boolean;
  describedBy?: string;
  onValueChange: (value: string | null) => void;
};

export function Select({
  id,
  label,
  value,
  options,
  placeholder,
  disabled,
  invalid,
  describedBy,
  onValueChange,
}: SelectProps) {
  const items = [{ value: null, label: placeholder }, ...options];
  return (
    <Primitive.Root
      items={items}
      value={value}
      onValueChange={onValueChange}
      disabled={disabled}
    >
      <Primitive.Trigger
        id={id}
        aria-label={label}
        aria-invalid={invalid}
        aria-describedby={describedBy}
        className="wk-select-trigger"
      >
        <Primitive.Value />
        <Primitive.Icon>
          <ChevronDownIcon aria-hidden="true" />
        </Primitive.Icon>
      </Primitive.Trigger>
      <Primitive.Portal>
        <Primitive.Positioner
          sideOffset={8}
          alignItemWithTrigger={false}
          className="wk-positioner"
        >
          <Primitive.Popup className="wk-select-popup">
            <Primitive.List>
              <Primitive.Group>
                {options.map((option) => (
                  <Primitive.Item
                    key={option.value}
                    value={option.value}
                    disabled={option.disabled}
                    className="wk-select-item"
                  >
                    <Primitive.ItemText>{option.label}</Primitive.ItemText>
                    <Primitive.ItemIndicator>
                      <CheckIcon aria-hidden="true" />
                    </Primitive.ItemIndicator>
                  </Primitive.Item>
                ))}
              </Primitive.Group>
            </Primitive.List>
          </Primitive.Popup>
        </Primitive.Positioner>
      </Primitive.Portal>
    </Primitive.Root>
  );
}

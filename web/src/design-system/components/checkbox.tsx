import { Checkbox as Primitive } from "@base-ui/react/checkbox";
import { CheckIcon, MinusIcon } from "lucide-react";

export function Checkbox(props: Primitive.Root.Props) {
  return (
    <Primitive.Root className="wk-checkbox" {...props}>
      <Primitive.Indicator>
        {props.indeterminate ? (
          <MinusIcon aria-hidden="true" />
        ) : (
          <CheckIcon aria-hidden="true" />
        )}
      </Primitive.Indicator>
    </Primitive.Root>
  );
}

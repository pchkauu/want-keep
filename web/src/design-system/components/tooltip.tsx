import { Tooltip as Primitive } from "@base-ui/react/tooltip";

export const TooltipProvider = Primitive.Provider;
export const Tooltip = Primitive.Root;
export const TooltipTrigger = Primitive.Trigger;
export function TooltipContent(props: Primitive.Popup.Props) {
  return (
    <Primitive.Portal>
      <Primitive.Positioner sideOffset={8} className="wk-positioner">
        <Primitive.Popup className="wk-tooltip" {...props} />
      </Primitive.Positioner>
    </Primitive.Portal>
  );
}

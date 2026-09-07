import { Popover as Primitive } from "@base-ui/react/popover";

export const Popover = Primitive.Root;
export const PopoverTrigger = Primitive.Trigger;
export const PopoverTitle = Primitive.Title;
export const PopoverDescription = Primitive.Description;
export const PopoverClose = Primitive.Close;
export function PopoverContent({ children, ...props }: Primitive.Popup.Props) {
  return (
    <Primitive.Portal>
      <Primitive.Positioner sideOffset={8} className="wk-positioner">
        <Primitive.Popup className="wk-popover" {...props}>
          {children}
        </Primitive.Popup>
      </Primitive.Positioner>
    </Primitive.Portal>
  );
}

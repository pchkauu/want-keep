import { Dialog as Primitive } from "@base-ui/react/dialog";
import { XIcon } from "lucide-react";
import { Button } from "./button";

export const Dialog = Primitive.Root;
export const DialogTrigger = Primitive.Trigger;
export const DialogClose = Primitive.Close;
export type DialogContentProps = Omit<
  Primitive.Popup.Props,
  "className" | "title"
> & {
  title: string;
  description: string;
  closeLabel: string;
  layout?: "dialog" | "sheet";
};

export function DialogContent({
  title,
  description,
  closeLabel,
  layout = "dialog",
  children,
  ...props
}: DialogContentProps) {
  return (
    <Primitive.Portal>
      <Primitive.Backdrop className="wk-backdrop" />
      <Primitive.Popup
        {...props}
        className={layout === "sheet" ? "wk-dialog wk-sheet" : "wk-dialog"}
      >
        <header className="wk-dialog-header">
          <div>
            <Primitive.Title className="wk-dialog-title">
              {title}
            </Primitive.Title>
            <Primitive.Description className="wk-description">
              {description}
            </Primitive.Description>
          </div>
          <Primitive.Close
            render={<Button variant="ghost" size="icon" />}
            aria-label={closeLabel}
          >
            <XIcon aria-hidden="true" />
          </Primitive.Close>
        </header>
        {children}
      </Primitive.Popup>
    </Primitive.Portal>
  );
}

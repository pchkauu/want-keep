import { AlertDialog as Primitive } from "@base-ui/react/alert-dialog";

export const AlertDialog = Primitive.Root;
export const AlertDialogTrigger = Primitive.Trigger;
export const AlertDialogClose = Primitive.Close;
export function AlertDialogContent({
  title,
  description,
  children,
  ...props
}: Omit<Primitive.Popup.Props, "title" | "className"> & {
  title: string;
  description: string;
}) {
  return (
    <Primitive.Portal>
      <Primitive.Backdrop className="wk-backdrop" />
      <Primitive.Popup {...props} className="wk-dialog">
        <Primitive.Title className="wk-dialog-title">{title}</Primitive.Title>
        <Primitive.Description className="wk-description">
          {description}
        </Primitive.Description>
        {children}
      </Primitive.Popup>
    </Primitive.Portal>
  );
}

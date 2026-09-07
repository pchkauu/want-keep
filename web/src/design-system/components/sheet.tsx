import {
  Dialog,
  DialogTrigger,
  DialogClose,
  DialogContent,
  type DialogContentProps,
} from "./dialog";

export {
  Dialog as Sheet,
  DialogTrigger as SheetTrigger,
  DialogClose as SheetClose,
};
export function SheetContent(props: Omit<DialogContentProps, "layout">) {
  return <DialogContent {...props} layout="sheet" />;
}

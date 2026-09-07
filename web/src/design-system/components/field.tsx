import { Field as Primitive } from "@base-ui/react/field";
import { Form as PrimitiveForm } from "@base-ui/react/form";
import type { ComponentProps } from "react";
import { cn } from "../class-names";

export const Form = PrimitiveForm;
export function Field({
  className,
  ...props
}: Omit<Primitive.Root.Props, "className"> & { className?: string }) {
  return <Primitive.Root {...props} className={cn("wk-field", className)} />;
}
export function FieldLabel(props: Primitive.Label.Props) {
  return <Primitive.Label className="wk-label" {...props} />;
}
export function FieldDescription(props: Primitive.Description.Props) {
  return <Primitive.Description className="wk-description" {...props} />;
}
export function FieldError(props: Primitive.Error.Props) {
  return <Primitive.Error className="wk-field-error" {...props} />;
}
export function FieldGroup({ className, ...props }: ComponentProps<"div">) {
  return <div {...props} className={cn("wk-field-group", className)} />;
}
export function FieldSet({ className, ...props }: ComponentProps<"fieldset">) {
  return <fieldset {...props} className={cn("wk-fieldset", className)} />;
}
export function FieldLegend(props: ComponentProps<"legend">) {
  return <legend className="wk-label" {...props} />;
}

import { Input as Primitive } from "@base-ui/react/input";
import { cn } from "../class-names";

export type InputProps = Omit<Primitive.Props, "className"> & {
  className?: string;
};
export function Input({ className, ...props }: InputProps) {
  return <Primitive {...props} className={cn("wk-input", className)} />;
}

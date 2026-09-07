import { Button as Primitive } from "@base-ui/react/button";
import type { VariantProps } from "class-variance-authority";
import { cn } from "../class-names";
import { buttonVariants } from "./button-variants";

export type ButtonProps = Omit<Primitive.Props, "className"> &
  VariantProps<typeof buttonVariants> & { className?: string };

export function Button({
  className,
  variant,
  size,
  type = "button",
  ...props
}: ButtonProps) {
  return (
    <Primitive
      {...props}
      type={type}
      className={cn(buttonVariants({ variant, size }), className)}
    />
  );
}

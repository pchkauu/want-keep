import type { ReactNode } from "react";
import { Button, type ButtonProps } from "./button";
import { cn } from "../class-names";

export function ActionTile({
  icon,
  children,
  className,
  ...props
}: Omit<ButtonProps, "variant" | "size"> & { icon: ReactNode }) {
  return (
    <Button
      {...props}
      variant="secondary"
      className={cn("wk-action-tile", className)}
    >
      <span aria-hidden="true">{icon}</span>
      <span>{children}</span>
    </Button>
  );
}

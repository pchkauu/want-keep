import type { ComponentProps } from "react";
import { cn } from "../class-names";

export function Surface({
  variant = "default",
  prominent = false,
  className,
  ...props
}: ComponentProps<"div"> & {
  variant?: "default" | "raised" | "accent";
  prominent?: boolean;
}) {
  return (
    <div
      {...props}
      className={cn("wk-surface", className)}
      data-variant={variant}
      data-prominent={prominent || undefined}
    />
  );
}

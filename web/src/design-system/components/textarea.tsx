import type { ComponentProps } from "react";
import { cn } from "../class-names";

export function Textarea({ className, ...props }: ComponentProps<"textarea">) {
  return (
    <textarea {...props} className={cn("wk-input wk-textarea", className)} />
  );
}

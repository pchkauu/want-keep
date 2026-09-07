import type { ComponentProps } from "react";
import { cn } from "../class-names";
import type { FeedbackTone } from "./feedback-tone";

export function Badge({
  tone = "neutral",
  className,
  ...props
}: ComponentProps<"span"> & { tone?: FeedbackTone }) {
  return (
    <span {...props} className={cn("wk-badge", className)} data-tone={tone} />
  );
}

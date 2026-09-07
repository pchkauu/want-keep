import type { ReactNode } from "react";

export function Empty({
  title,
  description,
  action,
}: {
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <div className="wk-empty">
      <h3>{title}</h3>
      <p className="wk-description">{description}</p>
      {action}
    </div>
  );
}

import type { ReactNode } from "react";

export function ListItem({
  leading,
  title,
  description,
  trailing,
}: {
  leading?: ReactNode;
  title: string;
  description: string;
  trailing?: ReactNode;
}) {
  return (
    <div className="wk-list-item">
      {leading && <div className="wk-list-leading">{leading}</div>}
      <div className="wk-list-copy">
        <p>{title}</p>
        <p className="wk-description">{description}</p>
      </div>
      {trailing}
    </div>
  );
}

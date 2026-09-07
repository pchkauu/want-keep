import type { ComponentProps } from "react";

export function Table({
  label,
  children,
  ...props
}: ComponentProps<"table"> & { label: string }) {
  return (
    <div
      role="region"
      aria-label={label}
      tabIndex={0}
      className="wk-table-region"
    >
      <table className="wk-table" {...props}>
        {children}
      </table>
    </div>
  );
}
export function TableHead(props: ComponentProps<"th">) {
  return <th scope="col" {...props} />;
}
export function TableCell(props: ComponentProps<"td">) {
  return <td {...props} />;
}

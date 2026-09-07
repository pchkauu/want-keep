import type { ReactNode } from "react";

export type SidebarItem = {
  id: string;
  label: string;
  href: string;
  icon?: ReactNode;
};
export function Sidebar({
  label,
  items,
  currentId,
  footer,
}: {
  label: string;
  items: readonly SidebarItem[];
  currentId: string;
  footer?: ReactNode;
}) {
  return (
    <nav className="wk-sidebar" aria-label={label}>
      <ul>
        {items.map((item) => (
          <li key={item.id}>
            <a
              href={item.href}
              aria-current={currentId === item.id ? "page" : undefined}
              className="wk-nav-link"
            >
              {item.icon}
              <span>{item.label}</span>
            </a>
          </li>
        ))}
      </ul>
      {footer && <div className="wk-sidebar-footer">{footer}</div>}
    </nav>
  );
}

export function Skeleton({ label }: { label: string }) {
  return (
    <div className="wk-skeleton" role="status" aria-label={label}>
      <span />
      <span />
      <span />
    </div>
  );
}

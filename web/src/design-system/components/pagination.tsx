import { ArrowLeftIcon, ArrowRightIcon } from "lucide-react";
import { Button } from "./button";

export type PaginationProps = {
  label: string;
  previousLabel: string;
  nextLabel: string;
  summary: string;
  hasPrevious: boolean;
  hasNext: boolean;
  busy?: boolean;
  onPrevious: () => void;
  onNext: () => void;
};
export function Pagination({
  label,
  previousLabel,
  nextLabel,
  summary,
  hasPrevious,
  hasNext,
  busy,
  onPrevious,
  onNext,
}: PaginationProps) {
  return (
    <nav className="wk-pagination" aria-label={label}>
      <Button
        variant="secondary"
        disabled={!hasPrevious || busy}
        onClick={onPrevious}
      >
        <ArrowLeftIcon aria-hidden="true" data-icon="inline-start" />
        {previousLabel}
      </Button>
      <span>{summary}</span>
      <Button variant="secondary" disabled={!hasNext || busy} onClick={onNext}>
        {nextLabel}
        <ArrowRightIcon aria-hidden="true" data-icon="inline-end" />
      </Button>
    </nav>
  );
}

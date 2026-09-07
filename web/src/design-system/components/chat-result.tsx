import type { ReactNode } from "react";
import { Badge } from "./badge";
import type { FeedbackTone } from "./feedback-tone";

export function ChatResult({
  author,
  status,
  title,
  explanation,
  tone = "neutral",
  details,
  actions,
}: {
  author: string;
  status: string;
  title: string;
  explanation: string;
  tone?: FeedbackTone;
  details?: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <article className="wk-chat-result">
      <header>
        <span className="wk-description">{author}</span>
        <Badge tone={tone}>{status}</Badge>
      </header>
      <h3>{title}</h3>
      <p className="wk-description">{explanation}</p>
      {details}
      {actions && <footer>{actions}</footer>}
    </article>
  );
}

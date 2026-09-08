import { useState } from "react";
import { Button } from "@/design-system/components/button";
import { useLocale } from "@/locales/locale";

export function RecoveryCodes({
  codes,
  onDone,
}: {
  codes: readonly string[];
  onDone: () => void;
}) {
  const { t } = useLocale();
  const [copyState, setCopyState] = useState<"copied" | "copyFailed">();
  return (
    <section className="recovery-codes" aria-labelledby="recovery-codes-title">
      <h2 id="recovery-codes-title">{t("codesTitle")}</h2>
      <p>{t("codesDescription")}</p>
      <ol>
        {codes.map((code) => (
          <li key={code}>
            <code>{code}</code>
          </li>
        ))}
      </ol>
      <div className="access-actions">
        <Button
          variant="secondary"
          onClick={() => {
            void navigator.clipboard.writeText(codes.join("\n")).then(
              () => setCopyState("copied"),
              () => setCopyState("copyFailed"),
            );
          }}
        >
          {t("copyCodes")}
        </Button>
        <Button
          variant="secondary"
          onClick={() => {
            const url = URL.createObjectURL(
              new Blob([codes.join("\n") + "\n"], { type: "text/plain" }),
            );
            const link = document.createElement("a");
            link.href = url;
            link.download = "want-keep-recovery-codes.txt";
            link.click();
            setTimeout(() => URL.revokeObjectURL(url), 1000);
          }}
        >
          {t("downloadCodes")}
        </Button>
      </div>
      {copyState && <p role="status">{t(copyState)}</p>}
      <Button onClick={onDone}>{t("savedCodes")}</Button>
    </section>
  );
}

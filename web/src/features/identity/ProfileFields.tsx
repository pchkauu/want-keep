import { Field, FieldLabel } from "@/design-system/components/field";
import { Input } from "@/design-system/components/input";
import { Select } from "@/design-system/components/select";
import { useLocale } from "@/locales/locale";
import { assets, type Asset } from "./model";

export function ProfileFields({
  name,
  asset,
  onName,
  onAsset,
  disabled,
}: {
  name: string;
  asset: Asset;
  onName: (name: string) => void;
  onAsset: (asset: Asset) => void;
  disabled: boolean;
}) {
  const { t } = useLocale();
  return (
    <>
      <Field>
        <FieldLabel>{t("yourName")}</FieldLabel>
        <Input
          required
          maxLength={2000}
          value={name}
          onValueChange={onName}
          disabled={disabled}
          autoComplete="name"
        />
      </Field>
      <Field>
        <FieldLabel>{t("reportingAsset")}</FieldLabel>
        <Select
          label={t("reportingAsset")}
          placeholder={t("reportingAsset")}
          value={asset}
          options={assets.map((value) => ({ value, label: value }))}
          disabled={disabled}
          onValueChange={(value) => {
            if (assets.includes(value as Asset)) onAsset(value as Asset);
          }}
        />
      </Field>
    </>
  );
}

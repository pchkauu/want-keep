import { Checkbox } from "./checkbox";

export type LegendItem = {
  id: string;
  label: string;
  valueLabel: string;
  marker: "solid" | "dashed" | "dotted";
  visible: boolean;
};
export function ChartLegend({
  label,
  items,
  onVisibleChange,
}: {
  label: string;
  items: readonly LegendItem[];
  onVisibleChange: (id: string, visible: boolean) => void;
}) {
  return (
    <fieldset className="wk-legend">
      <legend className="wk-label">{label}</legend>
      {items.map((item) => (
        <label key={item.id} className="wk-legend-item">
          <Checkbox
            checked={item.visible}
            onCheckedChange={(visible) => onVisibleChange(item.id, visible)}
          />
          <span
            className="wk-legend-marker"
            data-marker={item.marker}
            aria-hidden="true"
          />
          <span>{item.label}</span>
          <span className="wk-exact-amount">{item.valueLabel}</span>
        </label>
      ))}
    </fieldset>
  );
}

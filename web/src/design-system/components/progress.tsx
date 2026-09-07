import { Progress as Primitive } from "@base-ui/react/progress";

export function Progress({
  label,
  value,
  description,
}: {
  label: string;
  value: number | null;
  description: string;
}) {
  return (
    <Primitive.Root value={value} className="wk-progress">
      <Primitive.Label className="wk-label">{label}</Primitive.Label>
      <Primitive.Track className="wk-progress-track">
        <Primitive.Indicator className="wk-progress-indicator" />
      </Primitive.Track>
      <p className="wk-description">{description}</p>
    </Primitive.Root>
  );
}

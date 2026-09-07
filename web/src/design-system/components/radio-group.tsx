import { RadioGroup as Group } from "@base-ui/react/radio-group";
import { Radio } from "@base-ui/react/radio";

export function RadioGroup(props: Group.Props) {
  return <Group className="wk-radio-group" {...props} />;
}
export function RadioItem(props: Radio.Root.Props) {
  return (
    <Radio.Root className="wk-radio" {...props}>
      <Radio.Indicator className="wk-radio-dot" />
    </Radio.Root>
  );
}

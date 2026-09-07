import { ToggleGroup as Group } from "@base-ui/react/toggle-group";
import { Toggle } from "@base-ui/react/toggle";

export function ToggleGroup(props: Group.Props) {
  return <Group className="wk-toggle-group" {...props} />;
}
export function ToggleItem(props: Toggle.Props) {
  return <Toggle className="wk-toggle-item" {...props} />;
}

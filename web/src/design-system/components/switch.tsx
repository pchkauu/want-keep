import { Switch as Primitive } from "@base-ui/react/switch";

export function Switch(props: Primitive.Root.Props) {
  return (
    <Primitive.Root className="wk-switch" {...props}>
      <Primitive.Thumb className="wk-switch-thumb" />
    </Primitive.Root>
  );
}

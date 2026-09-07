import { Tabs as Primitive } from "@base-ui/react/tabs";

export const Tabs = Primitive.Root;
export function TabsList(props: Primitive.List.Props) {
  return <Primitive.List className="wk-tabs-list" {...props} />;
}
export function TabsTrigger(props: Primitive.Tab.Props) {
  return <Primitive.Tab className="wk-tab" {...props} />;
}
export function TabsContent(props: Primitive.Panel.Props) {
  return <Primitive.Panel className="wk-tab-panel" {...props} />;
}

import { Avatar as Primitive } from "@base-ui/react/avatar";

export function Avatar({
  name,
  initials,
  src,
}: {
  name: string;
  initials: string;
  src?: string;
}) {
  return (
    <Primitive.Root className="wk-avatar" aria-label={name}>
      {src && <Primitive.Image src={src} alt="" />}
      <Primitive.Fallback>{initials}</Primitive.Fallback>
    </Primitive.Root>
  );
}

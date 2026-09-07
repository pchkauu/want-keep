import { Toast as Primitive } from "@base-ui/react/toast";
import { XIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Button } from "./button";

export function Toaster({
  children,
  closeLabel,
}: {
  children: ReactNode;
  closeLabel: string;
}) {
  return (
    <Primitive.Provider limit={3} timeout={8000}>
      {children}
      <ToastList closeLabel={closeLabel} />
    </Primitive.Provider>
  );
}

function ToastList({ closeLabel }: { closeLabel: string }) {
  const { toasts } = Primitive.useToastManager();
  return (
    <Primitive.Portal>
      <Primitive.Viewport className="wk-toast-viewport">
        {toasts.map((toast) => (
          <Primitive.Root key={toast.id} toast={toast} className="wk-toast">
            <Primitive.Content>
              <div>
                <Primitive.Title className="wk-label" />
                <Primitive.Description className="wk-description" />
              </div>
              <Primitive.Close
                render={<Button variant="ghost" size="icon" />}
                aria-label={closeLabel}
                aria-hidden={false}
              >
                <XIcon aria-hidden="true" />
              </Primitive.Close>
            </Primitive.Content>
          </Primitive.Root>
        ))}
      </Primitive.Viewport>
    </Primitive.Portal>
  );
}

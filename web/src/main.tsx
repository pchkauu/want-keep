import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { RouterProvider } from "react-router/dom";
import { createApplication } from "@/app/router";
import { ApplicationServices } from "@/app/services";
import { RouteContext } from "@/navigation/route-context";
import "@/styles/global.css";

const root = document.getElementById("root");

if (!root) {
  throw new Error("Want Keep application root is missing");
}

const reactRoot = createRoot(root);
const invitationToken = RouteContext.takeInvitation(
  window.location,
  window.history,
);

if (import.meta.env.DEV && window.location.pathname === "/__design/tokens") {
  void import("@/design-system/preview/TokenPreview").then(
    ({ TokenPreview }) => {
      reactRoot.render(
        <StrictMode>
          <TokenPreview />
        </StrictMode>,
      );
    },
  );
} else if (
  import.meta.env.DEV &&
  window.location.pathname === "/__design/components"
) {
  void import("@/design-system/catalog/ComponentCatalog").then(
    ({ ComponentCatalog }) => {
      reactRoot.render(
        <StrictMode>
          <ComponentCatalog />
        </StrictMode>,
      );
    },
  );
} else {
  reactRoot.render(
    <StrictMode>
      <RouterProvider
        router={createApplication(new ApplicationServices(invitationToken))}
      />
    </StrictMode>,
  );
}

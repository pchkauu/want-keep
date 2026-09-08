import { createBrowserRouter, Navigate } from "react-router";
import { App, LoginRoute, InviteRoute, ProtectedShell } from "./App";
import { SetupPage, RecoveryPage, SecurityPage } from "@/features/identity";
import { RouteFailure } from "./RouteFailure";
import { UnavailablePage } from "./UnavailablePage";
import type { ApplicationServices } from "./services";
import { LocaleProvider } from "@/locales/LocaleProvider";
import { RouteLoading } from "./RouteLoading";

export function createApplication(services: ApplicationServices) {
  return createBrowserRouter([
    {
      element: <App services={services} />,
      errorElement: <RouteFailure />,
      hydrateFallbackElement: (
        <LocaleProvider controller={services.locale}>
          <RouteLoading />
        </LocaleProvider>
      ),
      children: [
        { path: "/login", element: <LoginRoute /> },
        { path: "/setup", element: <SetupPage /> },
        { path: "/invite", element: <InviteRoute /> },
        { path: "/recovery", element: <RecoveryPage /> },
        {
          element: <ProtectedShell />,
          children: [
            { index: true, element: <Navigate to="/overview" replace /> },
            {
              path: "/overview",
              lazy: async () => ({
                Component: (await import("./StartTrackingPage"))
                  .StartTrackingPage,
              }),
            },
            {
              path: "/onboarding",
              lazy: async () => ({
                Component: (await import("./StartTrackingPage")).OnboardingPage,
              }),
            },
            ...[
              "/accounts",
              "/plan",
              "/analytics",
              "/chat",
              "/connections",
              "/notifications",
            ].map((path) => ({ path, element: <UnavailablePage /> })),
            {
              path: "/settings",
              element: <Navigate to="/settings/security" replace />,
            },
            { path: "/settings/security", element: <SecurityPage /> },
            { path: "*", element: <UnavailablePage notFound /> },
          ],
        },
      ],
    },
  ]);
}

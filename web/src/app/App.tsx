import { useEffect } from "react";
import { Outlet, ScrollRestoration } from "react-router";
import {
  IdentityContext,
  useIdentity,
  LoginPage,
  InvitePage,
} from "@/features/identity";
import { LocaleProvider } from "@/locales/LocaleProvider";
import {
  ServicesContext,
  useServices,
  type ApplicationServices,
} from "./services";
import { SessionBoundary } from "./SessionBoundary";
import { DesktopShell } from "./DesktopShell";
import "./desktop-access.css";

function Root() {
  const services = useServices();
  const state = useIdentity();
  useEffect(() => services.identity.start(), [services]);
  useEffect(() => {
    if (state.status === "active" && state.member)
      services.locale.profile(state.member.locale);
  }, [services, state.status, state.member]);
  return (
    <>
      <Outlet />
      <ScrollRestoration />
    </>
  );
}
export function LoginRoute() {
  const services = useServices();
  const { member } = useIdentity();
  const destination =
    services.returnTo && services.returnTo.userId === member?.userId
      ? services.returnTo.path
      : "/overview";
  return <LoginPage destination={destination} />;
}
export function InviteRoute() {
  const services = useServices();
  return (
    <InvitePage
      initialToken={services.invitationToken}
      consumed={services.consumeInvitation}
    />
  );
}
export function App({ services }: { services: ApplicationServices }) {
  return (
    <ServicesContext.Provider value={services}>
      <LocaleProvider controller={services.locale}>
        <IdentityContext.Provider value={services.identity}>
          <Root />
        </IdentityContext.Provider>
      </LocaleProvider>
    </ServicesContext.Provider>
  );
}
export function ProtectedShell() {
  const { member } = useIdentity();
  return (
    <SessionBoundary>
      <DesktopShell key={member?.userId} />
    </SessionBoundary>
  );
}

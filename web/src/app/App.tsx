export function App() {
  return (
    <main className="foundation-shell bg-background text-foreground">
      <div className="foundation-shell__content">
        <img
          className="foundation-shell__logo"
          src="/brand/logo_512px.svg"
          width="455"
          height="512"
          alt=""
        />
        <p className="foundation-shell__eyebrow text-accent-readable">
          One place. All your money.
        </p>
        <h1>
          WANT <span>KEEP</span>
        </h1>
        <div className="foundation-shell__signal" aria-hidden="true">
          <span />
          <span />
          <span />
        </div>
        <p className="foundation-shell__status">Foundation build</p>
        <p className="foundation-shell__notice">
          Product workflows are intentionally unavailable in this build.
        </p>
      </div>
    </main>
  );
}

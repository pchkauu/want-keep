export function RouteFailure() {
  return (
    <main className="access-wait bg-background">
      <h1>Want Keep</h1>
      <p lang="ru">Не удалось открыть страницу. Вернитесь ко входу.</p>
      <p lang="en">This page could not be opened. Return to sign in.</p>
      <a className="access-link" href="/login">
        Войти / Sign in
      </a>
    </main>
  );
}

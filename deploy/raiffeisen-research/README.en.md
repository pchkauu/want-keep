# HTTPS for Raiffeisen research

[Русский](README.md)

This directory contains the initial `want-keep.tech` domain configuration for [task-0.2](../../spec/001-want-keep-mvp/tasks/task-0.2.md). It is research infrastructure, not the Want Keep application, an OAuth handler or a working bank connector. The target Go and Docker Compose application architecture is unchanged.

## Deployed configuration

Verified on 2026-09-07 on the owner-provided VPS: Ubuntu 26.04.1 LTS, nginx 1.28.3 from the Ubuntu repository, Certbot 4.0.0. The owner reports a German location. Ports 80/443 were free before installation; SSH and existing services were not changed.

| File | Purpose |
| --- | --- |
| `nginx-http.conf` | Initial HTTP-01 challenge; other requests return 503 until certificate issuance |
| `nginx-https.conf` | Active HTTPS configuration after certificate issuance |
| `reload-nginx` | Configuration validation and reload after successful certificate renewal |

VPS source files are in `/opt/want-keep/raiffeisen-research/`; the active configuration is `/etc/nginx/sites-available/want-keep.tech`, symlinked from `sites-enabled`. The ACME webroot is `/var/lib/want-keep-acme`. Certbot keeps the certificate and private key under its standard `/etc/letsencrypt/` directory; the key is excluded from the project directory. The deploy hook is installed at `/etc/letsencrypt/renewal-hooks/deploy/want-keep-nginx` with mode 0755; `certbot.timer` is active.

| Check | Result |
| --- | --- |
| DNS and HTTPS with certificate chain and hostname verification | Passed, including a Mac request without overriding DNS |
| Certificate expiry | 2026-12-05 23:54:51 UTC; renewal is checked independently |
| `GET /_health` | 204: only the HTTPS infrastructure works; it does not report bank import or application health |
| `GET /api/v1/connections/raiffeisen/callback` | 503: authorization codes are neither accepted nor processed |
| HTTP → HTTPS | 308 to the same path without query parameters, including synthetic code/state |
| HTTPS headers | no-store, no-referrer, nosniff, restrictive CSP |
| `nginx -t` | Passed before reload |
| `certbot renew --dry-run --cert-name want-keep.tech --run-deploy-hooks` | Passed; simulated renewal and the deploy hook ran |

Access logging is disabled and request error logging goes to `/dev/null` so future OAuth parameters do not enter request logs. This deliberately limits diagnostics in the initial configuration; the full application must add safe events without codes or tokens. Do not enable ordinary callback logging when replacing the placeholder. Pages have no third-party resources. Unknown HTTPS paths return 404; unknown SNI is rejected.

## Redeployment and recovery

Before changing anything, check enabled nginx sites, occupied ports, the certificate and configuration equality with the expected version. Do not overwrite unknown configuration. These files target the dedicated domain, not an arbitrary existing server.

On a new server, first install nginx and Certbot from the signed OS repository, enable `nginx-http.conf` and create the ACME webroot. Then issue the certificate:

```sh
certbot certonly --webroot --webroot-path /var/lib/want-keep-acme \
  --domain want-keep.tech --cert-name want-keep.tech \
  --non-interactive --agree-tos --register-unsafely-without-email --key-type ecdsa
```

The ACME account was created without email; certificate email reminders are not configured. If issuance has an unknown outcome, inspect `certbot certificates` before attempting another issuance. Replace the active file with `nginx-https.conf` only after the certificate exists, then run `nginx -t` and `systemctl reload nginx`. Preserve the previous file; restore it before reload if validation fails. Install the deploy hook, check the timer and run one simulated renewal.

During initial setup the stock nginx symlink was preserved as `/opt/want-keep/raiffeisen-research/default-enabled.before`; the pre-TLS HTTP configuration was preserved as `nginx-http.active.before-tls.conf`. Rolling back a specific change requires current-state inspection: do not restore the stock site over later changes. Restoring the HTTP configuration disables TLS availability after reload, so it is an emergency option, not normal renewal.

## Next step

The owner issued the initial Refresh token [in RBO](https://developer.raiffeisen.ru/docs/howToStart/tokens/howToGetTokensInOnlineBank). `probe.py` performed the refresh grant and two account GETs from Mac; `statements.py` retrieved two historical XML reports. The new token set is in a private local directory, outside Git. The initial Refresh token has already been replaced by the bank: subsequent commands use `tokens.json` as the source of truth, not the bootstrap file.

This initial issuance does not require a working callback and does not replace future application authorization. Code Flow remains disabled until state/PKCE/nonce, owner-session and connection-version checks are implemented under the [contract](../../spec/001-want-keep-mvp/contracts.en.md). BLK-02 remains open for the reasons in [evidence](../../spec/001-want-keep-mvp/evidence/raiffeisen.en.md).

## Diagnostic commands

Python standard library is used only for research; this is neither a Python service nor the Go connector implementation. The state directory must belong to the current user with mode 0700; result/token files are created with 0600. Credential paths are flags; values never go through arguments, environment or stdout. API calls verify TLS, disable HTTP redirects and ignore proxy environment. The read allowlist covers only accounts, the token endpoint and camt.052/053 report APIs; report payloads are validated separately.

Before running, the operator sets the variables below to their private paths. Refresh changes the token and is not required before every read. The example interval is illustrative; do not generate another report for an already recorded job.

```sh
python3 deploy/raiffeisen-research/probe.py refresh --state-dir "$RAIF_STATE" \
  --client-id-file "$RAIF_CLIENT_ID_FILE" --client-secret-file "$RAIF_CLIENT_SECRET_FILE" \
  --refresh-token-file "$RAIF_INITIAL_REFRESH_FILE"
python3 deploy/raiffeisen-research/probe.py accounts --state-dir "$RAIF_STATE"
python3 deploy/raiffeisen-research/statements.py check --state-dir "$RAIF_STATE" \
  --from-date 2026-08-01 --to-date 2026-08-31 --accounts-evidence "$RAIF_ACCOUNTS_EVIDENCE"
```

For historical statements use `check` → `create` → `status` → `file`, without automatic polling. `--kind camt-052` requires from/to equal to the current Europe/Moscow date; default camt-053 requires a completed interval. The intraday request for 2026-09-07 returned 404 no-statements; this is not a zero balance.

`refresh-state.json` is persisted before exchange. Timeout/unexpected response/save failure blocks retry; the original response, if received, is retained privately. Reports persist a marker before POST; later `create` for that interval is refused. The operator reconciles retained responses and bank state before recovery; do not automatically delete markers. A saved reportId allows separate status/file reads. `tokens.json` is one atomic complete set, not three independently replaced files. A nonblocking file lock protects state. This is local research storage, not the household application's final encrypted secret store.

`vps_accounts.py` performs only one account GET through SSH alias `want-keep.tech`. It needs separate approval to pass access/id tokens into that server's memory; refresh/client secret are not sent. SSH stdin/stdout are captured locally, argv contains no secrets, and the remote process writes no files. The SSH host key is verified; the password is supplied through external SSH_ASKPASS without placing its value in commands. The owner approved this run, which returned HTTP 200 and matched the Mac account. Ongoing credentials are not installed on the VPS.

```sh
python3 -m unittest discover -s deploy/raiffeisen-research -p 'test_*.py'
```

18 synthetic tests passed: rotation and reuse of saved tokens, unknown outcomes, malformed responses, locking, header injection, allowlist/redirect, report completion, rejection linked to its response, unknown status, rejection of credential storage inside Git and VPS transfer through stdin only. This does not prove the bank's full lifecycle, XSD compliance or application readiness.

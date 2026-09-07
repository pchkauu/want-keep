# Task-1.5 — protecting secrets and attachments

Result: backend/API for encrypted secrets, household uploads and safe PNG previews. Verified scope is D-46 and backend portions of AC-015/022/048/050/060/068/078/087/090. Task-1.6 is not a blocker: existing membership/sessions are used; invitations remain separate.

## Implementation and operational handoff

`privacy/cryptobox` owns AES-256-GCM/keyrings; `connections/access` owns permissions; `connections/credentials` handles plaintext immediately before the adapter. Attachment domain/application own upload identity, access and validation; files/processor/storage/delivery are outer boundaries. Shared HTTP protection lives in delivery/http/security; WebAuthn remains in identity. Migration 006 adds metadata, ciphertext, grants and immutable privacy audit; migrations 001–005 are preserved.

From backend, the operator runs `go run ./cmd/privacy-keygen --purpose attachments --output /private/path/attachment-keyring`, separately with `--purpose connections`. The CLI neither prints nor overwrites keys. The process owner creates a 0700 object directory. API loads paths from `WANT_KEEP_ATTACHMENT_KEYRING`, `WANT_KEEP_CONNECTION_KEYRING`, `WANT_KEEP_ATTACHMENT_DIRECTORY`, `WANT_KEEP_DOCUMENT_PROCESSOR_SOCKET`; key values never enter argv/environment. Missing resources disable only dependent functions. Restart reads require previous key IDs; backups must preserve keyrings separately from ciphertext without losing old keys.

`go run ./cmd/privacy-maintenance` removes at most 100 old pending names per invocation using the same attachment paths. Task-8.1 schedules it; keys/directories retain owner/permission checks. Publication uses atomic non-replacing hard links and fsync instead of overwriting rename. Metadata originalReady follows the original file; accepted follows all previews. Failures/lost replies recover through the original uploadId without financial effects. Accepted-document deletion and bulk rotation are not introduced.

`deploy/document-processor/compose.yaml` specifies mandatory isolation. Production API needs the private socket; processor never gets object storage/keyrings/DB. The image builds qpdf/Poppler from verified sources and copies only runtime libraries, fonts and licenses into scratch; no installation tools or network are present. HarfBuzz/Qt/GLib/CPP/curl/NSS/GPGME/boost and GUI tests are disabled. Poppler only rasterizes. The actual package and dynamic-library inventories are retained in `/usr/share/want-keep/licenses/`.

## Dependencies

| Component | Pin and license |
|---|---|
| Go/base | Go 1.26.5 trixie, digest `sha256:f1a132429b98724a904e9b3bdbaed399d8f923203c3e5170e6def66d0a7cc04c`; Debian main/security snapshot `20260901T000000Z` |
| qpdf | [12.4.1](https://github.com/qpdf/qpdf/releases/tag/v12.4.1), SHA-256 `f045aa277be2356ff53a89a8622945958291177d2483afc20ede7c8a8cd3873c`; [Apache-2.0](https://github.com/qpdf/qpdf/blob/v12.4.1/LICENSE.txt) (legacy Artistic-2.0 text retained), NOTICE and license texts in image |
| Poppler | [26.09.0](https://poppler.freedesktop.org/poppler-26.09.0.tar.xz), SHA-256 `8059eadb6805340768f138c465b57f8164c92b4a0773c37ef031ea6c0d987b2e`; GPL-2.0-or-later, COPYING in image |
| WebP | `golang.org/x/image v0.45.0`, BSD-3-Clause; checksum in go.sum |
| PostgreSQL tests | 17.11, digest `sha256:67f41722b7a8cbdb868a44a4995c846eddfdc2973bccb291ce937dce88ad5675` |

For image distribution, task-8.1 preserves license texts, notices and applicable GPL corresponding-source obligations.

## Verification and evidence boundaries

Required commands: `make check`, `make test-integration AREA=privacy`, identity/storage integration suites, identity/storage race suites and `git diff --check`. Privacy suite creates isolated PostgreSQL/processor, checks runtime isolation through Docker inspect, executes real HTTP/SQL/file/processor scenarios with Go race detection and removes only its own temporary resources. Infrastructure build/start failure is an error, never a skip. The publication ledger and CI bind actually passed checks to the exact SHA.

Coverage includes four formats, 10/11 PDF pages, 10 MiB, MIME/damage/encryption/active content and dimensions; household reads and outsider denial; no-store/nosniff/CSP; replay, concurrent upload, restart, leases, file write/publication failures and SQL rollback/lost acknowledgements; owner/session/purpose grants, single-use consumption, ciphertext/AAD and persisted-job secret reads; disconnect during IO with quarantine and preserved financial history. Child-process unit checks simulate crashes/deadlines. A probe present only in the integration image triggers OOM in the processor cgroup, verifies kernel oom_kill, then restarts the container and repeats HTTP/lease scenarios. The production image does not contain the probe.

Remaining AC portions stay with their owners: AC-015/022 receipt extraction/matching, AC-048/087 live platforms and OAuth/MFA, AC-050/060 complete AI gateway and prompt-injection scenarios, AC-068/078/090 UI/household product actions, AC-072/088 notification delivery. Authorized accepted attachments may be AI input; secrets never are. Live banks, AI calls, Chrome/Arc, UI and production are not claimed as verified. SDD is Ready for development; operational readiness is not established.

Additional regressions cover indirect Launch through HTTP/DB/the pinned processor; a killed qpdf leaves the same uploadId pending, then healthy processing accepts it. Supervisor tests synchronize cancellation with file creation and check cleanup, the next document, restart and exclusive workspace ownership. The output limiter hides bytes.Buffer methods that could bypass size validation through io.Copy.


Integration with task-1.6 preserves its invitation/ownership policies and migration 005. The undeployed privacy migration is now 006; no production data is changed. Shared delivery retains one Origin/CSRF/session guard; connections load both household and secret purpose.

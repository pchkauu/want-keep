# Infrastructure and server budget research

[Русский](hosting.md)

Snapshot date: 2026-09-07, Europe/Moscow. Task: [task-0.9 / Issue #9](https://github.com/pchkauu/want-keep/issues/9). The owner's current VPS in Germany, public Timeweb Cloud pages/documentation and outbound public requests from the VPS were inspected. Secrets, IP addresses, real financial data and original responses with sensitive fields are not retained in Git.

**Research is complete with operational blockers.** The owner selected a 2 vCPU / 4 GB RAM application server in Germany and managed PostgreSQL in the same location. The verifiable target profile remains within the household $40/month limit under the conservative estimate. The current 1 vCPU, sub-1 GB VPS remains a research endpoint only. Provisioning, hardening, load measurement, backup/restore rehearsal and complete provider access were not performed; their owners are listed in HOST-B01–HOST-B06.

## Decision

| Component | MVP target | Basis and boundary |
| --- | --- | --- |
| Application server | Timeweb Cloud DE-50, Frankfurt: 2 vCPU, 4 GB RAM, 50 GB NVMe, up to 200 Mbps | Owner decision. The server runs reverse proxy/web, Go API and worker, and one sequential Playwright collector. PostgreSQL does not run on the VPS. The plan is an upper sizing choice pending cgroup, disk and collector measurements. |
| Database | Timeweb Cloud managed PostgreSQL, Germany: 1 vCPU, 2 GB RAM, 20 GB, single node | Hundreds of transactions per month do not justify an HA cluster. 2 GB permits separate application, migration and backup roles. task-8.1 pins a supported major after driver/migration compatibility checks; the service offers PostgreSQL 14–18. Managed does not mean HA: replication requires a separate, more expensive profile. |
| Network | Free private VPC/BGP network in one German location; public IPv4 on the VPS only | Disable managed PostgreSQL public IP. The application uses a private endpoint with TLS verification. Only 80/443 are public; SSH is restricted to an administrative allowlist. |
| Containers | Docker Compose for web/reverse proxy, API, worker and isolated collector | Production Compose does not own PostgreSQL. Development and integration environments retain an isolated PostgreSQL container. Run the collector as non-root with one worker and CPU/RAM/process/network limits. |
| Backups | Hourly outbound pull from the Mac; logical managed PostgreSQL dump plus immutable attachments | Do not rely on the VPS or provider backup as the only independent copy. Mark success only after the Mac verifies the manifest and checksums. |

The provider data-centre page lists Frankfurt sites and advertises Tier III/99.98%; this is a provider claim rather than a measured Want Keep SLA: [data centres](https://timeweb.cloud/docs/nashi-data-centry). Private networks are available in Germany: [VPC](https://timeweb.cloud/services/vpc), [adding services to a BGP network](https://timeweb.cloud/docs/vpc/managing-bgp-networks/adding-services-to-bgp). Managed DB public IP can be disabled: [public-IP management](https://timeweb.cloud/docs/dbaas/dbaas-manage/public-ip-access).

## Cost estimate

All amounts are a 2026-09-07 public-price snapshot with VAT as stated on the page. The owner's current server costs RUB 800/month; its invoice and tariff/IP/tax breakdown were not inspected.

| Item | Annual-price mode | Conservative, no discount |
| --- | ---: | ---: |
| VPS DE-50, 2 vCPU / 4 GB / 50 GB | RUB 1,530/month | RUB 1,700/month |
| Managed PostgreSQL, 1 vCPU / 2 GB / 20 GB | RUB 790/month | RUB 877.78/month |
| One public IPv4 for the VPS | RUB 200/month | RUB 200/month |
| **Total** | **RUB 2,520/month** | **RUB 2,777.78/month** |
| Conservative + 10% reserve | — | **RUB 3,055.56/month** |

D-15 comparison uses the [official Bank of Russia rate](https://www.cbr.ru/currency_base/daily/?UniDbQuery.Posted=True&UniDbQuery.To=05.09.2026) effective 2026-09-05: 86.5857 RUB/USD. The annual-price total is about $29.10/month; the conservative total with reserve is about $35.29/month, leaving about $4.71 below the $40 limit. OpenAI up to $50 and domain/certificate or domain renewal are separate. No public DB IP, paid replication or paid server backup is included.

Sources: [German server pricing](https://timeweb.cloud/services/cloud-servers?location=de), [managed databases](https://timeweb.cloud/services/dbaas), [managed DB creation](https://timeweb.cloud/docs/dbaas/dbaas-create), [PostgreSQL connection](https://timeweb.cloud/docs/dbaas/postgresql/connect-to-database). Annual mode requires a 12-month term and applies a 10% discount; that commitment is not treated as accepted, so the conservative no-discount column is the admission basis. task-8.1 must re-read the tariff and FX rate before purchase because this snapshot does not guarantee a future price.

## Current-server snapshot

Read-only SSH inspection ran on 2026-09-07. Exact IPs and credential values are not published.

| Area | Confirmed observation | Implication |
| --- | --- | --- |
| Location/environment | KVM VPS at Timeweb Cloud; route geolocation is Frankfurt am Main, Germany. The network registry country differs and does not determine physical location by itself | Germany is supported by combined runtime/provider/routing evidence; this is a dated snapshot, not an SLA |
| OS | Ubuntu 26.04.1 LTS, x86_64, NTP synchronization enabled | Playwright supports Ubuntu 26.04 x86_64; the production image must still match the pinned Playwright version |
| Resources | 1 AMD EPYC-Rome vCPU; 889 MiB RAM with about 687 MiB available idle; no swap; roughly 14 GiB root filesystem with about 11 GiB free | Below the selected production profile; do not admit the full MVP/collector on this size without measurement |
| Runtime | Docker, Compose, Go, Node.js and PostgreSQL absent | The application and DB are not deployed; research HTTPS is not Want Keep runtime |
| Public services | nginx listens on 80/443, sshd on 22; Zabbix agent binds `0.0.0.0:10050` | Provider/host firewall, allowlisting and a decision on Zabbix protection/need precede financial data |
| Updates | unattended upgrades and apt timers enabled; no pending packages or reboot-required observed | Positive snapshot state, not a substitute for a patch/monitoring policy |
| TLS | HTTP redirects to HTTPS; Let's Encrypt certificate for `want-keep.tech` is valid through 2026-12-05 | DNS/TLS work for the research domain; renewal/expiry alerts remain task-8.1 |

Provider limitations must be considered before deployment: Frankfurt has no IPv6, paid DDoS protection is not available in every location, and some outbound ports are blocked: [cloud-server limitations](https://timeweb.cloud/docs/cloud-servers/limitations). CPU/RAM can scale, disk can only grow, and tariff changes restart a server: [server configuration](https://timeweb.cloud/docs/cloud-servers/manage-servers/server-configuration).

## Reachability from Germany

Only DNS/TLS/HTTP reachability to public addresses was checked, without bank sessions, API keys or financial requests. HTTP 401/403/404 can establish a responding destination but does not establish a working future adapter.

| Destination | Current VPS result | What it establishes |
| --- | --- | --- |
| OpenAI `/v1/models` | HTTP 401 without a key | DNS/TLS/route reachable; account, model, limits and Responses API unverified |
| Bank of Russia, Frankfurter, CoinGecko | HTTP 200 | Selected public rate endpoints were reachable at check time |
| Raiffeisen developer endpoint | HTTP 200 | Public developer endpoint reachable; live API evidence belongs to task-0.2 |
| Bybit public market time | HTTP 200 | Public endpoint reachable; private Funding/Earn/P2P unverified by this probe |
| Aifory public site | HTTP 200 | DNS/TLS/public web only |
| EMCD API root | HTTP 404 | Destination responds; correct API route and authorization unverified |
| Ozon Finance public site | HTTP 403 | Destination responds and rejects this anonymous request; application access is not established |
| Alfa-Bank | System resolver failed for tested domains; DoH returned addresses, but forced connection failed TLS verification | Safe reachability is not established. TLS bypass is forbidden; HOST-B04 remains open |

Provider firewall state cannot be inferred from local `ss` or TCP-MTR: a destination response may be an accept or a reject. External exposure of 10050 and provider ACL therefore remain unverified until task-8.1 reads the control-plane rules.

## Security gate before financial data

The current server uses root/password SSH, permits password authentication, has no active host firewall and exposes a monitoring-agent listener on all interfaces. The local operator-password file outside the repository had mode `0644`; its content and path are not recorded in documentation. These facts make the endpoint unsuitable for financial data.

Before any secrets are connected, task-8.1 must:

1. create an unprivileged administrator, verify key-only access and a recovery path, then disable root/password access and X11 forwarding;
2. restrict the operator credential file to its owner (`0600`) or replace it with a key/secret manager;
3. apply provider and host firewalls: expose 80/443, restrict 22 to an administrative allowlist, and remove, close or mutually authenticate 10050;
4. keep application/provider/DB secrets outside Git, separate DB roles, redact logs and exclude financial content;
5. prohibit public managed PostgreSQL, use private VPC and verify TLS hostname/CA;
6. pin image digests/versions and apply non-root collector, cgroup/pid/network allowlist, rate-limit and observability controls.

These are requirements for a subsequent task, not completed server changes. The VPS configuration was not changed during research.

## Backup and recovery

The Mac runs an hourly `launchd` job and makes an outbound connection. No inbound Mac port is opened. A dedicated unprivileged server principal permits only the backup-export command, without TTY/agent/port forwarding. The application creates a consistent cutoff and manifest; over the private network, the VPS streams a logical `pg_dump` from managed PostgreSQL and an inventory of immutable attachments. The Mac encrypts the set, verifies sizes/checksums and only then atomically marks it complete. An interrupted set never replaces the last valid one.

The recovery key stays outside the VPS, DB and sole backup: the working copy resides in macOS Keychain and an emergency duplicate in an independent password manager or offline store. task-8.2 selects and pins the exact backup tool; this research fixes the contract without adding a dependency.

MVP retention is 48 hourly, 30 daily, 8 weekly and 12 monthly complete points. The deduplicated repository has a 20 GiB cap; warn at 15 GiB or when Mac free space falls below 25 GiB. Never delete the last complete set. If policy does not fit, report failure and require storage expansion instead of silently deleting the safety point.

The inspected Mac had about 46 GiB free; available system commands did not establish FileVault/target-volume encryption. Before enablement, preflight free space, encryption, recovery-key access and a test restore. The planning assumption is at most 500 attachment files/month averaging 2 MiB, about 12 GiB of raw attachment flow per year; this is unverified and is not an upload limit. Measure actual sizes before enabling retention.

Do not retain a permanent full backup on the VPS. Plan the 50 GB disk as up to 15 GB OS/images/logs, 20 GB live attachments, 5 GB temporary workspace and at least 10 GB headroom. Warn at 70%, critical at 80%; exceeding the threshold stops temporary-file creation without deleting source data. Prefer streaming the DB dump.

RPO ≤1 hour applies only while the Mac is reachable and the last set completes checksum verification. While offline, display increasing age and the real potential-loss window. RTO ≤4 hours remains a task-8.3 runtime criterion. A managed/provider backup can aid operational recovery but does not replace the independent Mac copy.

## Evidence ledger

| ID | Status and source | Finding |
| --- | --- | --- |
| HOST-E01 | confirmed, Issue #9 and SDD at research revision | Limits, mandatory components and research boundaries recorded |
| HOST-E02 | confirmed, owner statement + read-only SSH, 2026-09-07 | Current German server reachable and costs RUB 800/month; credentials omitted |
| HOST-E03 | confirmed, OS/kernel/cpu/memory/disk/runtime commands | Current 1 vCPU/<1 GB/14 GB profile is below the production target |
| HOST-E04 | confirmed, sshd/firewall/listeners/update inspection | Hardening required before financial data |
| HOST-E05 | confirmed, DNS/HTTP/TLS/certificate inspection | Research domain works; product runtime is not established |
| HOST-E06 | confirmed, provider/routing evidence + [Timeweb data centres](https://timeweb.cloud/docs/nashi-data-centry) | Frankfurt is an allowed location; provider SLA unmeasured |
| HOST-E07 | confirmed, [German VPS pricing](https://timeweb.cloud/services/cloud-servers?location=de) | DE-50 2/4/50 selected; dated price needs readback before purchase |
| HOST-E08 | confirmed, [DBaaS pricing](https://timeweb.cloud/services/dbaas) and [creation contract](https://timeweb.cloud/docs/dbaas/dbaas-create) | Single-node 1/2/20 selected; public IP unnecessary |
| HOST-E09 | confirmed, [VPC](https://timeweb.cloud/services/vpc), [PostgreSQL connection](https://timeweb.cloud/docs/dbaas/postgresql/connect-to-database) | App and DB can share private network; endpoint not yet created |
| HOST-E10 | confirmed, anonymous live probes from VPS | OpenAI/rates and several platform public endpoints respond with stated boundaries |
| HOST-E11 | confirmed gap, resolver/TLS probes | Safe Alfa reachability not established; certificate verification was not bypassed |
| HOST-E12 | confirmed/unknown, local Mac disk/security preflight | About 46 GiB free; target-volume encryption unverified |
| HOST-E13 | confirmed, [Playwright system requirements](https://playwright.dev/docs/next/intro), [Docker guidance](https://playwright.dev/docs/next/docker), [CI workers](https://playwright.dev/docs/ci) | Ubuntu 26.04 x86_64 supported; crawling requires non-root/seccomp, version match and worker=1 |
| HOST-E14 | confirmed arithmetic using dated public prices and CBR rate | $35.29 conservative ceiling with reserve is below $40 |
| HOST-E15 | design decision + explicit capacity assumptions | Pull/manifest/retention/RPO contract is implementation-ready; runtime backup unproven |

## Blockers and handoff

| ID | Status | Required action and owner |
| --- | --- | --- |
| HOST-B01 | OPEN | task-8.1: purchase/create selected VPS, managed PostgreSQL and private VPC only under separate authorization; read back final invoice and endpoints |
| HOST-B02 | OPEN | task-8.1: complete the security gate, including SSH/firewalls/Zabbix/secrets/TLS/logging, before storing financial data |
| HOST-B03 | OPEN | task-8.1: measure Compose/cgroups, sequential Playwright peak RAM/CPU, disk growth and degradation; revise within $40 if insufficient |
| HOST-B04 | OPEN | task-0.10 and task-4.1: establish a safe Alfa DNS/TLS/route contract from the target VPS; the Alfa adapter remains blocked |
| HOST-B05 | OPEN | task-8.2/task-8.3: verify Mac encryption/capacity, implement backup and measure restore/RPO/RTO; the research design alone does not pass AC-057/AC-058 |
| HOST-B06 | OPEN | task-8.1/task-8.2: after DB creation verify private endpoint, CA/hostname, roles, `pg_dump` compatibility and actual provider backup policy |

task-0.9 completes configuration selection and research and unblocks the infrastructure part of task-0.10. The complete MVP remains **Not Ready**. AC-048/AC-056/AC-057/AC-058 are not claimed as passed; they require production load, hardening, observability and backup/restore rehearsal.

## Verification

Completed: read-only server inventory, service/listener/firewall/SSH/update/TLS checks, public endpoint probes from the VPS, official pricing/docs review, budget arithmetic, Mac capacity preflight and RU/EN semantic review. No server setting, subscription or financial operation changed.

Documentation commands: `make docs-check`, `python3 -m unittest discover -s spec/001-want-keep-mvp/tools -p 'test_*.py'`, `git diff --check`. They verify the SDD, not runtime.

Not completed: target VPS/DB/VPC provisioning, invoice readback, authenticated platform/OpenAI requests, safe Alfa route, Playwright load/soak, provider-firewall readback, hardening, backup implementation and restore rehearsal. Reasons are the research-only scope, absent product runtime and separate authorization required for infrastructure purchases/changes.

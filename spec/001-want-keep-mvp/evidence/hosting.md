# Инфраструктура и бюджет сервера: исследование

[English](hosting.en.md)

Дата среза: 2026-09-07, Europe/Moscow. Задача: [task-0.9 / Issue #9](https://github.com/pchkauu/want-keep/issues/9). Проверены текущий VPS владельца в Германии, публичные страницы и документация Timeweb Cloud, а также исходящие публичные запросы с VPS. Секреты, IP-адреса, реальные финансовые данные и исходные ответы с чувствительными полями не сохранены в Git.

**Исследование завершено с эксплуатационными блокерами.** Владелец выбрал сервер приложения 2 vCPU / 4 ГБ RAM в Германии и managed PostgreSQL в той же локации. Проверяемый целевой профиль укладывается в семейный лимит $40/месяц даже по консервативной смете. Текущий VPS с 1 vCPU и менее 1 ГБ RAM остаётся только исследовательским стендом. Provisioning, hardening, нагрузочные измерения, backup/restore rehearsal и полный доступ к платформам не выполнялись; их владельцы указаны в HOST-B01–HOST-B06.

## Решение

| Компонент | Целевая конфигурация MVP | Основание и граница |
| --- | --- | --- |
| Сервер приложения | Timeweb Cloud DE-50, Франкфурт: 2 vCPU, 4 ГБ RAM, 50 ГБ NVMe, до 200 Мбит/с | Выбор владельца. На сервере работают reverse proxy/web, Go API и worker, один последовательный Playwright collector. PostgreSQL на VPS не размещается. Тариф — верхняя оценка до измерения cgroups, диска и collector. |
| База данных | Timeweb Cloud managed PostgreSQL, Германия: 1 vCPU, 2 ГБ RAM, 20 ГБ, один узел | Сотни операций в месяц не требуют HA-кластера. 2 ГБ позволяют отдельные роли приложения, миграций и backup. Конкретную поддерживаемую major-версию фиксирует task-8.1 после проверки драйвера и миграций; сервис предлагает PostgreSQL 14–18. Managed не означает HA: репликация требует отдельного более дорогого профиля. |
| Сеть | Бесплатная приватная VPC/BGP-сеть в одной немецкой локации; публичный IPv4 только у VPS | У managed PostgreSQL отключается публичный IP. Соединение приложения с БД идёт по private endpoint с TLS-проверкой. Публичны только 80/443; SSH — по административному allowlist. |
| Контейнеры | Docker Compose для web/reverse proxy, API, worker и изолированного collector | Production Compose не владеет PostgreSQL. Локальные и integration-окружения сохраняют изолированный PostgreSQL-контейнер. Collector запускается без root, с одной worker и ограничениями CPU/RAM/process/network. |
| Копии | Почасовой исходящий pull с Mac; logical dump managed PostgreSQL и неизменяемые вложения | Не полагаться на VPS или provider backup как на единственную независимую копию. Успех ставится только после локальной проверки manifest и checksums. |

Источник по дата-центрам указывает площадки Франкфурта и Tier III/99.98% как заявление провайдера, а не измеренный SLA Want Keep: [дата-центры](https://timeweb.cloud/docs/nashi-data-centry). Private networks доступны в Германии: [VPC](https://timeweb.cloud/services/vpc), [добавление сервисов в BGP-сеть](https://timeweb.cloud/docs/vpc/managing-bgp-networks/adding-services-to-bgp). Публичный IP managed DB можно отключить: [управление публичным IP](https://timeweb.cloud/docs/dbaas/dbaas-manage/public-ip-access).

## Смета

Все суммы ниже — снимок публичных цен на 2026-09-07 с указанным на странице НДС. Реальный текущий сервер стоит 800 ₽/месяц со слов владельца; счёт и разложение этой суммы на тариф/IP/налог не читались.

| Статья | Годовой тариф | Консервативно без скидки |
| --- | ---: | ---: |
| VPS DE-50, 2 vCPU / 4 ГБ / 50 ГБ | 1 530 ₽/мес. | 1 700 ₽/мес. |
| Managed PostgreSQL, 1 vCPU / 2 ГБ / 20 ГБ | 790 ₽/мес. | 877.78 ₽/мес. |
| Один публичный IPv4 для VPS | 200 ₽/мес. | 200 ₽/мес. |
| **Итого** | **2 520 ₽/мес.** | **2 777.78 ₽/мес.** |
| Консервативно + 10% резерв | — | **3 055.56 ₽/мес.** |

Для сравнения с D-15 использован [официальный курс Банка России](https://www.cbr.ru/currency_base/daily/?UniDbQuery.Posted=True&UniDbQuery.To=05.09.2026) с effective date 2026-09-05: 86.5857 RUB/USD. Годовая цена равна примерно $29.10/месяц; консервативный вариант с резервом — примерно $35.29/месяц, оставляя около $4.71 до лимита $40. OpenAI до $50, домен и возможное продление сертификата/домена считаются отдельно. Публичный IP БД, платная репликация и платный server backup в профиль не входят.

Источники: [тарифы серверов в Германии](https://timeweb.cloud/services/cloud-servers?location=de), [managed databases](https://timeweb.cloud/services/dbaas), [создание managed DB](https://timeweb.cloud/docs/dbaas/dbaas-create), [подключение PostgreSQL](https://timeweb.cloud/docs/dbaas/postgresql/connect-to-database). Годовой режим требует 12-месячного срока и даёт скидку 10%; обязательство не считается уже принятым, поэтому основой допуска служит консервативная колонка без скидки. Тариф и курс обязательно перечитываются task-8.1 перед заказом: этот снимок не гарантирует будущую цену.

## Снимок текущего сервера

Read-only SSH-проверка выполнена 2026-09-07. Точные IP и значения credentials не публикуются.

| Область | Подтверждённое наблюдение | Следствие |
| --- | --- | --- |
| Локация и среда | KVM VPS Timeweb Cloud; геолокация маршрута — Frankfurt am Main, Germany. Registry country сети отличается и сам по себе не определяет физическую площадку | Германия подтверждена совокупностью runtime/provider/routing evidence; это датированный снимок, не SLA |
| ОС | Ubuntu 26.04.1 LTS, x86_64, синхронизация NTP включена | Playwright поддерживает Ubuntu 26.04 x86_64; production image всё равно должен совпадать с закреплённой версией Playwright |
| Ресурсы | 1 vCPU AMD EPYC-Rome; 889 MiB RAM, около 687 MiB доступно в idle; swap отсутствует; root filesystem около 14 GiB, около 11 GiB свободно | Ниже выбранного production-профиля; полный MVP и collector на этом размере не допущены без измерений |
| Runtime | Docker, Compose, Go, Node.js и PostgreSQL отсутствуют | Приложение и БД не развёрнуты; исследовательский HTTPS не является runtime Want Keep |
| Публичные сервисы | nginx слушает 80/443, sshd — 22; Zabbix agent привязан к `0.0.0.0:10050` | Перед данными нужны provider/host firewall, allowlist и решение по защищённости/необходимости Zabbix |
| Обновления | unattended upgrades и apt timers включены; pending packages и reboot-required не обнаружены | Положительное состояние среза, но не заменяет patch/monitoring policy |
| TLS | HTTP перенаправляет на HTTPS; сертификат Let's Encrypt для `want-keep.tech` действителен до 2026-12-05 | DNS/TLS работают для исследовательского домена; renewal/expiry alert остаются task-8.1 |

Провайдерские ограничения следует учитывать до развёртывания: во Франкфурте нет IPv6, платная DDoS-защита доступна не во всех локациях, некоторые исходящие порты блокируются: [ограничения cloud servers](https://timeweb.cloud/docs/cloud-servers/limitations). Масштабирование CPU/RAM возможно, диск только увеличивается, смена тарифа перезапускает сервер: [server configuration](https://timeweb.cloud/docs/cloud-servers/manage-servers/server-configuration).

## Достижимость из Германии

Проверены только DNS/TLS/HTTP до публичных адресов, без банковских сессий, API-ключей и финансовых запросов. HTTP 401/403/404 может подтвердить достижимость узла, но не работоспособность будущего адаптера.

| Назначение | Результат с текущего VPS | Что доказано |
| --- | --- | --- |
| OpenAI `/v1/models` | HTTP 401 без ключа | DNS/TLS/маршрут доступны; аккаунт, модель, лимиты и Responses API не проверены |
| Банк России, Frankfurter, CoinGecko | HTTP 200 | Выбранные публичные rate endpoints достижимы в момент проверки |
| Raiffeisen developer endpoint | HTTP 200 | Публичный developer endpoint достижим; live API evidence ведётся в task-0.2 |
| Bybit public market time | HTTP 200 | Публичный endpoint достижим; private Funding/Earn/P2P не проверены этим тестом |
| Aifory public site | HTTP 200 | Только DNS/TLS/public web |
| EMCD API root | HTTP 404 | Узел отвечает; правильный API route и авторизация не проверены |
| Ozon Finance public site | HTTP 403 | Узел отвечает и запрещает этот анонимный запрос; приложение не признано доступным |
| Альфа-Банк | Системный resolver не разрешил проверенные домены; DoH дал адреса, но принудительное соединение не прошло проверку TLS | Безопасная достижимость не подтверждена. Обход TLS запрещён; HOST-B04 остаётся открыт |

Provider firewall нельзя определить по локальному `ss` или TCP-MTR: ответ destination может быть как accept, так и reject. Поэтому внешний exposure 10050 и provider ACL остаются непроверенными до чтения панели/правил в task-8.1.

## Security gate до финансовых данных

Текущий сервер использует root/password SSH, password authentication разрешён, host firewall не активен, а monitoring agent слушает все интерфейсы. Локальный файл с операторским паролем вне репозитория имел режим `0644`; содержимое и путь не читались в документацию. Эти факты делают стенд непригодным для хранения финансовых данных.

task-8.1 до подключения секретов обязан:

1. создать непривилегированного администратора, проверить key-only вход и recovery path, затем отключить root/password и X11 forwarding;
2. ограничить файл операторского секрета владельцем (`0600`) либо заменить его ключом/менеджером секретов;
3. применить provider firewall и host firewall: публичны 80/443, 22 доступен только административному allowlist; 10050 удалить, закрыть или защитить взаимной аутентификацией;
4. хранить application/provider/DB secrets вне Git, разделить роли БД, редактировать логи и исключить финансовое содержимое;
5. запретить публичный managed PostgreSQL, включить private VPC и проверить TLS hostname/CA;
6. закрепить image digests/versions, non-root collector, cgroups/pids/network allowlist, rate limits и наблюдаемость.

Это требования следующей задачи, а не выполненные изменения сервера. В рамках исследования конфигурация VPS не менялась.

## Backup и восстановление

Mac запускает задачу каждый час через `launchd` и устанавливает исходящее соединение. Входящий порт на Mac не открывается. Выделенный непривилегированный server principal допускает только backup-export command без TTY/agent/port forwarding. Приложение создаёт согласованный cutoff и manifest; VPS по private network потоково читает logical `pg_dump` managed PostgreSQL и inventory неизменяемых вложений. Mac шифрует набор, проверяет размеры/checksums и только затем атомарно отмечает его complete. Прерванный набор не заменяет последний исправный.

Recovery key хранится отдельно от VPS, БД и единственной копии: рабочая копия — в macOS Keychain, аварийная — в независимом password manager или offline-хранилище. Конкретный backup tool и его закреплённая версия выбираются task-8.2; исследование фиксирует контракт, не добавляет зависимость.

Политика хранения MVP: 48 почасовых, 30 дневных, 8 недельных и 12 месячных полных точек. Дедуплицированный репозиторий ограничен 20 GiB; предупреждение — при размере 15 GiB либо свободном месте Mac менее 25 GiB. Последний complete set не удаляется. Если политика не помещается, система сообщает ошибку и требует увеличить хранилище вместо скрытого удаления сохранной точки.

На проверенном Mac около 46 GiB свободно; FileVault/шифрование целевого volume не удалось подтвердить доступными системными командами. До запуска нужен preflight свободного места, шифрования, доступа к recovery key и пробного restore. Оценка вложений для планирования — не более 500 файлов/месяц со средним размером 2 MiB, около 12 GiB сырого потока в год; это непроверенное предположение, не upload limit. Реальные размеры измеряются до включения retention.

На VPS не хранится полный постоянный backup. Диск 50 ГБ планируется так: до 15 ГБ ОС/images/logs, 20 ГБ активных вложений, 5 ГБ временного рабочего места и не менее 10 ГБ headroom. Warning — 70%, critical — 80%; превышение останавливает создание временных файлов, не удаляя source data. DB dump предпочтительно передаётся потоково.

RPO ≤ 1 часа действует только при доступном Mac и успешной checksum-проверке последнего набора. При offline возраст и фактическое окно потери растут и показываются пользователю. Цель RTO ≤ 4 часов остаётся runtime-критерием task-8.3. Managed/provider backup может ускорить операционное восстановление, но не заменяет независимую Mac-копию.

## Evidence ledger

| ID | Статус и источник | Вывод |
| --- | --- | --- |
| HOST-E01 | confirmed, Issue #9 и SDD на ревизии исследования | Лимиты, обязательные компоненты и границы исследования зафиксированы |
| HOST-E02 | confirmed, owner statement + read-only SSH, 2026-09-07 | Текущий сервер в Германии доступен и стоит 800 ₽/мес.; credentials не публикуются |
| HOST-E03 | confirmed, OS/kernel/cpu/memory/disk/runtime commands | Текущий 1 vCPU/<1 ГБ/14 ГБ профиль меньше production target |
| HOST-E04 | confirmed, sshd/firewall/listeners/update inspection | Требуется hardening до финансовых данных |
| HOST-E05 | confirmed, DNS/HTTP/TLS/certificate inspection | Исследовательский домен работает; продуктовый runtime не доказан |
| HOST-E06 | confirmed, provider/routing evidence + [Timeweb data centres](https://timeweb.cloud/docs/nashi-data-centry) | Франкфурт — разрешённая локация; provider SLA не измерен |
| HOST-E07 | confirmed, [German VPS pricing](https://timeweb.cloud/services/cloud-servers?location=de) | Выбран DE-50 2/4/50; цена датирована и требует readback перед заказом |
| HOST-E08 | confirmed, [DBaaS pricing](https://timeweb.cloud/services/dbaas) и [creation contract](https://timeweb.cloud/docs/dbaas/dbaas-create) | Выбран single-node 1/2/20; public IP не нужен |
| HOST-E09 | confirmed, [VPC](https://timeweb.cloud/services/vpc), [PostgreSQL connection](https://timeweb.cloud/docs/dbaas/postgresql/connect-to-database) | App и DB можно связать в одной private network; фактический endpoint ещё не создан |
| HOST-E10 | confirmed, live anonymous probes from VPS | OpenAI/rates и часть platform public endpoints отвечают с оговорёнными статусами |
| HOST-E11 | confirmed gap, resolver/TLS probes | Безопасная достижимость Alfa не подтверждена; проверка сертификата не обходилась |
| HOST-E12 | confirmed/unknown, local Mac disk/security preflight | Около 46 GiB свободно; шифрование volume не подтверждено |
| HOST-E13 | confirmed, [Playwright system requirements](https://playwright.dev/docs/next/intro), [Docker guidance](https://playwright.dev/docs/next/docker), [CI workers](https://playwright.dev/docs/ci) | Ubuntu 26.04 x86_64 поддерживается; crawling требует non-root/seccomp, version match и worker=1 |
| HOST-E14 | confirmed arithmetic using dated public prices and CBR rate | $35.29 conservative ceiling with reserve is below $40 |
| HOST-E15 | design decision + explicit capacity assumptions | Pull/manifest/retention/RPO contract готов к реализации; runtime backup не доказан |

## Блокеры и передача

| ID | Статус | Нужное действие и владелец |
| --- | --- | --- |
| HOST-B01 | OPEN | task-8.1: заказать/создать выбранные VPS, managed PostgreSQL и private VPC только после отдельной авторизации; проверить итоговый invoice и endpoint readback |
| HOST-B02 | OPEN | task-8.1: выполнить security gate, включая SSH/firewalls/Zabbix/secrets/TLS/logging, до помещения финансовых данных |
| HOST-B03 | OPEN | task-8.1: измерить Compose/cgroups, sequential Playwright peak RAM/CPU, disk growth и деградацию; при нехватке пересмотреть профиль внутри $40 |
| HOST-B04 | OPEN | task-0.10 и task-4.1: установить безопасный Alfa DNS/TLS/route contract из целевого VPS; без этого Alfa adapter заблокирован |
| HOST-B05 | OPEN | task-8.2/task-8.3: проверить Mac encryption/capacity, реализовать backup и измерить restore/RPO/RTO; исследовательская схема не проходит AC-057/AC-058 сама по себе |
| HOST-B06 | OPEN | task-8.1/task-8.2: после создания DB проверить private endpoint, CA/hostname, роли, `pg_dump` compatibility и фактическую provider backup policy |

task-0.9 завершает выбор и исследование конфигурации и разблокирует инфраструктурную часть task-0.10. Полный MVP остаётся **Not Ready**. AC-048/AC-056/AC-057/AC-058 не объявляются пройденными: для них требуются production load, hardening, наблюдаемость и backup/restore rehearsal.

## Проверка

Выполнены read-only server inventory, сервисы/listeners/firewall/SSH/update/TLS checks, публичные endpoint probes с VPS, официальный pricing/docs review, расчёт бюджета, Mac capacity preflight и RU/EN semantic review. Никакие настройки сервера, подписки или финансовые операции не менялись.

Команды документации: `make docs-check`, `python3 -m unittest discover -s spec/001-want-keep-mvp/tools -p 'test_*.py'`, `git diff --check`. Они проверяют SDD, но не runtime.

Не выполнены: provisioning целевого профиля/DB/VPC, invoice readback, authenticated platform/OpenAI requests, Alfa safe route, Playwright load/soak, provider firewall readback, hardening, backup implementation и restore rehearsal. Причина — исследовательский объём, отсутствие product runtime и необходимость отдельной авторизации на приобретение/изменение инфраструктуры.

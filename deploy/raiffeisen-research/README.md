# HTTPS для исследования Raiffeisen

[English](README.en.md)

Этот каталог содержит начальную конфигурацию домена `want-keep.tech` для [task-0.2](../../spec/001-want-keep-mvp/tasks/task-0.2.md). Это инфраструктура исследования, не приложение Want Keep, OAuth-обработчик или готовый банковский коннектор. Целевая архитектура приложения на Go и Docker Compose сохраняется.

## Что развёрнуто

Проверено 2026-09-07 на предоставленном владельцем VPS: Ubuntu 26.04.1 LTS, nginx 1.28.3 из репозитория Ubuntu, Certbot 4.0.0. Местоположение в Германии сообщено владельцем. До установки порты 80/443 были свободны; SSH и существующие сервисы не изменялись.

| Файл | Назначение |
| --- | --- |
| `nginx-http.conf` | Первоначальный HTTP-01 challenge; остальное отвечает 503 до выпуска сертификата |
| `nginx-https.conf` | Рабочая конфигурация HTTPS после выпуска сертификата |
| `reload-nginx` | Проверка конфигурации и reload после успешного продления сертификата |

На VPS исходники находятся в `/opt/want-keep/raiffeisen-research/`, активная конфигурация — `/etc/nginx/sites-available/want-keep.tech` с symlink в `sites-enabled`. ACME webroot — `/var/lib/want-keep-acme`. Certbot хранит сертификат и закрытый ключ в стандартном `/etc/letsencrypt/`; ключ не включается в каталог проекта. Deploy hook установлен как `/etc/letsencrypt/renewal-hooks/deploy/want-keep-nginx` с правами 0755; `certbot.timer` активен.

| Проверка | Результат |
| --- | --- |
| DNS и HTTPS с проверкой цепочки и имени сертификата | Успешно, включая запрос с Mac без подмены DNS |
| Срок сертификата | До 2026-12-05 23:54:51 UTC; продление проверяется независимо |
| `GET /_health` | 204: работает только HTTPS-инфраструктура, не банковский импорт и не приложение |
| `GET /api/v1/connections/raiffeisen/callback` | 503: код авторизации не принимается и не обрабатывается |
| HTTP → HTTPS | 308 на тот же путь без query-параметров, включая синтетические code/state |
| Заголовки HTTPS | no-store, no-referrer, nosniff, ограничивающая CSP |
| `nginx -t` | Успешно перед reload |
| `certbot renew --dry-run --cert-name want-keep.tech --run-deploy-hooks` | Успешно; тестовое продление и deploy hook выполнены |

Access log выключен, request error log направлен в `/dev/null`, чтобы будущие OAuth-параметры не попадали в журналы запросов. Это сознательное ограничение диагностики начальной конфигурации; полноценное приложение должно добавить безопасные события без кодов и токенов. Не включать обычное журналирование callback при замене заглушки. На страницах нет сторонних ресурсов. Неизвестные HTTPS-пути отвечают 404; неизвестный SNI отклоняется.

## Повторное развёртывание и восстановление

Перед изменением проверить активные сайты nginx, занятость портов, сертификат и совпадение конфигурации с ожидаемой версией. Не перезаписывать неизвестную конфигурацию. Эти файлы относятся к выделенному домену, не к произвольному существующему серверу.

На новом сервере сначала устанавливаются nginx и Certbot из подписанного репозитория ОС, включается `nginx-http.conf` и создаётся ACME webroot. Затем выпускается сертификат:

```sh
certbot certonly --webroot --webroot-path /var/lib/want-keep-acme \
  --domain want-keep.tech --cert-name want-keep.tech \
  --non-interactive --agree-tos --register-unsafely-without-email --key-type ecdsa
```

Аккаунт ACME создан без email; почтовые напоминания о сертификате не настроены. Повторный запуск issuance после неизвестного результата сначала требует `certbot certificates`, а не нового выпуска. Только после наличия сертификата заменить активный файл на `nginx-https.conf`, выполнить `nginx -t` и `systemctl reload nginx`. Сохранить предыдущий файл; при неуспешной проверке вернуть его до reload. Установить deploy hook, проверить timer и один раз выполнить dry-run продления.

При первоначальной настройке stock-ссылка nginx сохранена как `/opt/want-keep/raiffeisen-research/default-enabled.before`; HTTP-конфигурация перед включением TLS — как `nginx-http.active.before-tls.conf`. Откат конкретного изменения требует проверки текущего состояния: не восстанавливать stock-сайт поверх последующих изменений. Возврат HTTP-конфигурации отключает доступность TLS после reload, поэтому это аварийный вариант, а не обычное продление.

## Следующий шаг

Первоначальный Refresh-токен выпущен владельцем [в RBO](https://developer.raiffeisen.ru/docs/howToStart/tokens/howToGetTokensInOnlineBank). `probe.py` выполнил refresh grant и два GET счетов с Mac; `statements.py` получил две исторические XML-выписки. Новый комплект токенов находится в закрытом локальном каталоге, не в Git. Первоначальный Refresh-токен уже заменён банком: source of truth для последующих команд — `tokens.json`, не исходный bootstrap-файл.

Такой первоначальный выпуск не требует готового callback и не заменяет будущую авторизацию приложения. Code Flow остаётся выключен до реализации проверок state/PKCE/nonce, сессии владельца и версии подключения по [контракту](../../spec/001-want-keep-mvp/contracts.md). BLK-02 остаётся открыт по причинам в [evidence](../../spec/001-want-keep-mvp/evidence/raiffeisen.md).

## Диагностические команды

Python standard library используется только для исследования; это не Python-сервис и не реализация Go-коннектора. Каталог состояния должен принадлежать текущему пользователю и иметь mode 0700; файлы результатов/токенов создаются с 0600. Пути credentials передаются флагами, значения не передаются через аргументы, environment или stdout. API-запросы идут с проверкой TLS, без HTTP redirects и proxy environment. Read allowlist ограничен счетами, token endpoint и camt.052/053 report API; тела запросов отчётов проверяются отдельно.

Перед запуском оператор задаёт переменные ниже путями к своим закрытым файлам/каталогу. Команда refresh изменяет токен; она не нужна перед каждым чтением. Пример периода условный — не запускать генерацию повторно для уже сохранённого задания.

```sh
python3 deploy/raiffeisen-research/probe.py refresh --state-dir "$RAIF_STATE" \
  --client-id-file "$RAIF_CLIENT_ID_FILE" --client-secret-file "$RAIF_CLIENT_SECRET_FILE" \
  --refresh-token-file "$RAIF_INITIAL_REFRESH_FILE"
python3 deploy/raiffeisen-research/probe.py accounts --state-dir "$RAIF_STATE"
python3 deploy/raiffeisen-research/statements.py check --state-dir "$RAIF_STATE" \
  --from-date 2026-08-01 --to-date 2026-08-31 --accounts-evidence "$RAIF_ACCOUNTS_EVIDENCE"
```

Для исторической выписки последовательность `check` → `create` → `status` → `file`, без автоматического polling. `--kind camt-052` требует from/to текущего дня Europe/Moscow; по умолчанию camt-053 требует закрытый интервал. Запрос текущего дня 2026-09-07 вернул 404 no-statements; это не нулевой остаток.

`refresh-state.json` сохраняется до обмена. При timeout/неожиданном ответе/сбое сохранения повтор блокируется; оригинальный ответ, если получен, сохраняется приватно. У отчёта до POST сохраняется marker; последующий `create` этого интервала запрещён. Оператор сверяет сохранённые ответы и состояние банка перед восстановлением; автоматически удалять markers нельзя. Сохранённый reportId позволяет отдельно читать status/file. `tokens.json` — атомарный полный набор, а не три независимо заменяемых файла. Состояние защищено неблокирующей файловой блокировкой. Это локальное исследовательское хранение, не финальное зашифрованное хранилище семейного приложения.

`vps_accounts.py` выполняет только один GET счетов с SSH alias `want-keep.tech`. Он требует отдельного разрешения на передачу access/id токенов в память этого сервера; refresh/client secret не передаются. stdin/stdout SSH перехватываются локально, секретов в argv нет, удалённый процесс не пишет файлы. SSH host key проверяется; пароль подключается через внешний SSH_ASKPASS без значения в командах. Этот прогон разрешён владельцем и прошёл HTTP 200; счёт совпал с Mac. Постоянные credentials на VPS не установлены.

```sh
python3 -m unittest discover -s deploy/raiffeisen-research -p 'test_*.py'
```

18 синтетических тестов прошли: ротация и повтор из сохранённого набора, unknown outcome, malformed response, lock, header injection, allowlist/redirect, report completion, сохранение отказа со ссылкой на ответ, неизвестный статус, запрет хранения credentials внутри Git и передача в VPS только через stdin. Это не доказательство полного lifecycle банка, XSD или готовности приложения.

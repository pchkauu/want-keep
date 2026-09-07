# Task-1.5 — защита секретов и вложений

Результат: backend/API для зашифрованных секретов, семейных загрузок и безопасных PNG preview. Проверяемая область — D-46 и backend-части AC-015/022/048/050/060/068/078/087/090. Task-1.6 не блокирует работу: используются существующие membership/session, приглашения остаются отдельной задачей.

## Реализация и эксплуатационная передача

`privacy/cryptobox` владеет AES-256-GCM и keyring; `connections/access` — разрешениями, `connections/credentials` — plaintext непосредственно перед адаптером. Attachment domain/application управляют идентичностью загрузки, доступом и валидацией; files/processor/storage/delivery — внешние границы. Общая HTTP-защита вынесена в delivery/http/security, WebAuthn остаётся в identity. Миграция 005 добавляет метаданные, ciphertext, grants и неизменяемый privacy audit; история 001–004 сохранена.

Из backend оператор запускает `go run ./cmd/privacy-keygen --purpose attachments --output /private/path/attachment-keyring` и отдельно `--purpose connections`. CLI не печатает и не перезаписывает ключи. Каталог объектов создаётся владельцем процесса с 0700. API использует пути из `WANT_KEEP_ATTACHMENT_KEYRING`, `WANT_KEEP_CONNECTION_KEYRING`, `WANT_KEEP_ATTACHMENT_DIRECTORY`, `WANT_KEEP_DOCUMENT_PROCESSOR_SOCKET`; значения ключей не передаются через argv/environment. Отсутствие этих ресурсов отключает только зависимые функции. Для повторного чтения после рестарта нужны прежние key IDs; backup должен сохранять keyring отдельно от ciphertext, не теряя старые ключи.

`go run ./cmd/privacy-maintenance` удаляет до 100 старых pending-имён за запуск с теми же attachment-путями. Task-8.1 назначает расписание; ключи/каталог загружаются с проверками владельца и прав. Публикация использует атомарный hard link без перезаписи и fsync вместо заменяющего rename. Metadata originalReady подтверждается после файла; accepted — после всех preview. Ошибка/потеря ответа восстанавливаются повтором исходного uploadId, без финансового эффекта. Удаление принятых документов и массовая ротация не добавлены.

`deploy/document-processor/compose.yaml` задаёт обязательную изоляцию. Production API должен иметь приватный socket, но processor не получает каталог объектов/keyring/DB. Образ собирает qpdf/Poppler из проверенных исходников и переносит только runtime-библиотеки, шрифты и лицензии в scratch image; установочных инструментов и сети там нет. HarfBuzz/Qt/GLib/CPP/curl/NSS/GPGME/boost и тестовые GUI выключены. Poppler используется только для растеризации. Список фактических пакетов и динамических библиотек сохраняется внутри `/usr/share/want-keep/licenses/`.

## Зависимости

| Компонент | Закрепление и лицензия |
|---|---|
| Go/base | Go 1.26.5 trixie, digest `sha256:f1a132429b98724a904e9b3bdbaed399d8f923203c3e5170e6def66d0a7cc04c`; Debian main/security snapshot `20260901T000000Z` |
| qpdf | [12.4.1](https://github.com/qpdf/qpdf/releases/tag/v12.4.1), SHA-256 `f045aa277be2356ff53a89a8622945958291177d2483afc20ede7c8a8cd3873c`; [Apache-2.0](https://github.com/qpdf/qpdf/blob/v12.4.1/LICENSE.txt) (legacy Artistic-2.0 text retained), NOTICE и тексты в образе |
| Poppler | [26.09.0](https://poppler.freedesktop.org/poppler-26.09.0.tar.xz), SHA-256 `8059eadb6805340768f138c465b57f8164c92b4a0773c37ef031ea6c0d987b2e`; GPL-2.0-or-later, COPYING в образе |
| WebP | `golang.org/x/image v0.45.0`, BSD-3-Clause; checksum в go.sum |
| PostgreSQL tests | 17.11, digest `sha256:67f41722b7a8cbdb868a44a4995c846eddfdc2973bccb291ce937dce88ad5675` |

При распространении образа task-8.1 сохраняет тексты лицензий, notices и обязательства предоставления соответствующих исходников GPL.

## Проверки и пределы доказательств

Обязательные команды: `make check`, `make test-integration AREA=privacy`, identity/storage integration suites, identity/storage race suites и `git diff --check`. Privacy suite сама создаёт изолированные PostgreSQL/processor, проверяет runtime isolation через Docker inspect, запускает реальные HTTP/SQL/file/processor сценарии с Go race detector, затем удаляет только собственные временные ресурсы. Невозможность сборки/запуска инфраструктуры — ошибка, не skip. Ledger публикации и CI привязывают фактически пройденные проверки к точному SHA.

Покрытие включает четыре формата, 10/11 PDF страниц, 10 MiB, MIME/повреждения/шифрование/активное содержимое и размеры; семейное чтение и отказ посторонним; no-store/nosniff/CSP; повторы, конкурентную загрузку, restart, lease, ошибки записи/публикации файла и rollback/lost acknowledgement SQL; owner/session/purpose grants, одноразовое потребление, ciphertext/AAD и выдачу секрета по сохранённому job; отключение во время IO с quarantine и сохранением финансовой истории. Unit-проверки дочернего процесса моделируют crash/deadline. Отдельный probe только в integration image вызывает OOM в cgroup процессора, проверяет kernel oom_kill, затем контейнер перезапускается и HTTP/lease-сценарии выполняются повторно. Production image не содержит probe.

Остальные части AC остаются профильным задачам: AC-015/022 — распознавание/сопоставление чека, AC-048/087 — реальные платформы и OAuth/MFA, AC-050/060 — полный AI gateway и prompt-injection сценарий, AC-068/078/090 — UI/семейные продуктовые действия, AC-072/088 — доставка уведомлений. Разрешённое принятое вложение может быть AI input; секреты никогда не являются AI input. Не заявляются реальные банки, AI-вызовы, Chrome/Arc, UI и production. SDD — Ready for development; эксплуатационная готовность не подтверждена.

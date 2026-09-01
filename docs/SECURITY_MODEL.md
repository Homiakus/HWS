# HWS Security Model

Статус: Initial threat-model baseline.

## 1. Цель

HWS потенциально запускает приложения, открывает каталоги, управляет пользовательскими сервисами, подключается к удалённым системам и меняет desktop state. Поэтому он должен проектироваться как orchestration system с чёткими trust boundaries, а не как набор произвольных shell shortcuts.

## 2. Trust boundaries

```text
User input / UI
      ↓
IPC boundary
      ↓
hwsd application layer
      ↓
Axiom policy/orchestration
      ↓
integration adapters
      ↓
OS / desktop / network / remote systems
```

Отдельные недоверенные/частично доверенные источники:

- dynamic providers;
- workspace config;
- imported hierarchy definitions;
- Git repository metadata;
- remote server responses;
- external command output.

## 3. Principle of least privilege

`hwsd` работает как обычный пользовательский процесс.

Запрещено запускать весь daemon с root только потому, что одна будущая функция может требовать повышенных прав.

Если privileged operations станут необходимыми, создаётся отдельный минимальный helper с allowlisted API.

## 4. Command execution

По умолчанию command spec имеет форму:

```go
type CommandSpec struct {
    Executable string
    Args       []string
    WorkingDir string
    Env        map[string]string
}
```

Запрещён default path:

```text
user config → /bin/sh -c <string>
```

Shell mode, если когда-либо появится, должен быть отдельной явно опасной capability и policy.

## 5. Executable resolution

Правила:

- разрешение executable выполняется предсказуемо;
- resolved path может фиксироваться в diagnostic data;
- workspace не должен подменять PATH неявно;
- environment mutation ограничена policy;
- запуск из writable untrusted directories требует отдельного анализа.

## 6. Working directory and paths

Любые paths проходят normalization/validation.

Проверять:

- traversal;
- symlink semantics;
- existence/type where required;
- ownership/permissions where relevant;
- race между validation и use для чувствительных операций.

## 7. Secrets

Секреты не помещаются в:

- WorkspaceDefinition;
- Axiom history;
- обычные logs;
- diagnostics bundle;
- search index;
- UI status metadata.

Используется отдельный secret provider/OS keyring integration, когда это понадобится.

Axiom activity input не должен содержать секрет, если runtime/history/store может его сериализовать.

## 8. IPC

Требования:

- локальный versioned API;
- привязка к пользовательской сессии;
- peer identity validation в пределах возможностей выбранного transport;
- отсутствие remote listen by default;
- request size limits;
- cancellation/timeouts;
- structured errors;
- capability checks внутри daemon, а не доверие UI.

UI считается удобным клиентом, но не security authority.

## 9. Dynamic providers

Provider не получает автоматически право выполнять команды.

Начальная permission model:

```text
read project metadata
read filesystem scope
read git metadata
publish nodes
request predefined action
```

Произвольный provider-generated executable intent запрещён без отдельного permission model.

## 10. Workspace definitions

Config является данными, не кодом.

Workspace может ссылаться на зарегистрированные action kinds/adapters, но не должен автоматически получать произвольный доступ ко всем функциям ОС.

## 11. Destructive actions

Действия классифицируются:

- read-only;
- reversible/user-session-local;
- state-changing;
- destructive;
- privileged.

Класс влияет на:

- UI affordance;
- confirmation policy;
- retry policy;
- logging;
- required authorization/capability.

## 12. Resource ownership

Перед destructive lifecycle action HWS подтверждает ownership.

Нельзя завершать процесс только потому, что его имя совпало с ожидаемым приложением.

Identity evidence может включать:

- launch record;
- process ancestry;
- executable identity;
- desktop application ID;
- adapter-specific stable token;
- creation correlation.

PID один сам по себе недостаточен как долговременная identity.

## 13. Remote integrations

SSH/remote adapters должны:

- использовать системные/проверенные credential mechanisms;
- не отключать host verification по умолчанию;
- не логировать секреты;
- иметь явный remote target identity;
- различать read-only discovery и command execution.

## 14. Network/VPN actions

HWS не должен реализовывать скрытый универсальный privilege bypass.

Network operations проходят через системные supported interfaces/adapter и policy.

Смена маршрутизации/VPN должна быть видимой пользователю и иметь recovery semantics.

## 15. Axiom security boundary

Axiom обеспечивает rules/claims/runtime semantics, но не заменяет authorization.

Перед external activity HWS должен уже иметь application-level решение, что действие разрешено.

Claims полезны как дополнительная fail-closed защита, например:

```text
privileged action requires approved capability
release requires managed ownership
required identity confidence reached
```

## 16. Logs and diagnostics

Structured logging использует allowlist безопасных полей.

Не логировать целиком:

- environment;
- config blobs;
- remote responses;
- command output без ограничения;
- activity input/output, если они могут содержать чувствительные данные.

Diagnostics bundle проходит redaction и показывает пользователю, что включено в архив.

## 17. Update / supply chain

До release нужны:

- dependency vulnerability scanning;
- locked module versions;
- reproducible-ish build metadata;
- signed release strategy;
- checksum verification;
- documented update channel;
- no silent arbitrary code download by providers.

## 18. Plugin model

Полноценные plugins не входят в ранний MVP.

До появления plugin execution необходимо отдельное ADR, потому что in-process Go plugins, external processes и WASM имеют разные security/failure boundaries.

## 19. Threat scenarios

### T1. Malicious workspace config запускает shell command

Mitigation: config data model + allowlisted typed actions; no implicit shell.

### T2. HWS закрывает чужой editor process

Mitigation: ownership + identity evidence + shutdown policy.

### T3. UI compromised/buggy sends privileged action

Mitigation: daemon-side policy; UI not authority.

### T4. Retry дважды создаёт внешний ресурс

Mitigation: idempotency key + observe-before-create + post-action verification.

### T5. Provider injects fake system action

Mitigation: provider returns typed nodes/requests; capability/permission validation.

### T6. Secrets leak to history

Mitigation: secret provider boundary; no secret-bearing activity payloads/history.

### T7. Daemon restart causes destructive cleanup

Mitigation: fresh observed state + ownership revalidation before cleanup.

## 20. Security acceptance criteria для MVP

- [ ] `hwsd` не требует root.
- [ ] no implicit shell execution path.
- [ ] ownership enforced before process release.
- [ ] IPC local-only by default.
- [ ] dynamic providers cannot execute arbitrary commands.
- [ ] secrets absent from fixtures/history/log samples.
- [ ] Axiom external activities have explicit idempotency model.
- [ ] restart recovery revalidates observed state.
- [ ] diagnostics redaction tested.
- [ ] threat model пересматривается перед добавлением remote/network privileged features.
# HWS Domain Model

Статус: Proposed baseline для M0/M1.

## 1. Цель

Определить минимальную устойчивую модель HWS, независимую от GNOME, конкретного IPC и конкретного storage backend.

## 2. Идентификаторы

Все сущности, на которые могут ссылаться config/history/workspace, имеют стабильные типизированные идентификаторы.

```go
type NodeID string
type WorkspaceID string
type ResourceID string
type ActionID string
type ProviderID string
type OperationID string
```

Правила:

- display title не является ID;
- ID не зависит от позиции в дереве;
- rename/move узла не меняют ID;
- внешние raw PID/window handle не используются как стабильный ResourceID;
- runtime handles хранятся отдельно от domain identity.

## 3. Node

```go
type NodeKind string

const (
    NodeCategory NodeKind = "category"
    NodeProject  NodeKind = "project"
    NodeWorkspace NodeKind = "workspace"
    NodeAction   NodeKind = "action"
    NodeWidget   NodeKind = "widget"
    NodeQuery    NodeKind = "query"
)

type Node struct {
    ID       NodeID
    ParentID *NodeID
    Kind     NodeKind
    Title    string
    Subtitle string
    Order    int
    Target   TargetRef
    Source   NodeSource
    Metadata Metadata
}
```

### TargetRef

Узел может вести к:

- дочернему path;
- workspace;
- action;
- dynamic provider;
- информационному view.

TargetRef должен быть tagged union/typed representation, а не произвольной строкой.

## 4. Tree Snapshot

UI работает не с mutable pointers на живое дерево, а с versioned snapshot/projection.

```go
type TreeSnapshot struct {
    Revision string
    Root     NodeID
    Nodes    []Node
}
```

Требования:

- один `NodeID` встречается не более одного раза;
- parent существует, кроме root;
- cycles запрещены;
- child ordering детерминирован;
- snapshot immutable после публикации;
- revision меняется при семантическом изменении.

## 5. SelectionPath

```go
type SelectionPath struct {
    TreeRevision string
    Nodes        []NodeID
}
```

Инварианты:

- первый элемент — root или direct root child согласно UI projection contract;
- каждый следующий node является ребёнком предыдущего в данном snapshot;
- path не содержит duplicates/cycles;
- пустой path имеет явную семантику, а не случайный nil.

## 6. WorkspaceDefinition

```go
type WorkspaceDefinition struct {
    ID          WorkspaceID
    Version     string
    Title       string
    Resources   []ResourceSpec
    Layout      LayoutIntent
    Environment EnvironmentSpec
    Startup     StartupPolicy
    Shutdown    ShutdownPolicy
    Recovery    RecoveryPolicy
    Requirements CapabilityRequirements
}
```

WorkspaceDefinition — декларация намерения пользователя.

Она не должна содержать volatile данные вроде PID или window handle.

## 7. ResourceSpec

```go
type ResourceKind string

const (
    ResourceApplication ResourceKind = "application"
    ResourceTerminal    ResourceKind = "terminal"
    ResourceService     ResourceKind = "service"
    ResourceWindow      ResourceKind = "window"
    ResourceRemote      ResourceKind = "remote"
    ResourceNetwork     ResourceKind = "network"
    ResourceDocument    ResourceKind = "document"
    ResourceProvider    ResourceKind = "provider"
)

type RequirementLevel string

const (
    Required  RequirementLevel = "required"
    Preferred RequirementLevel = "preferred"
    Optional  RequirementLevel = "optional"
)

type ResourceSpec struct {
    ID          ResourceID
    Kind        ResourceKind
    Requirement RequirementLevel
    Desired     ResourceDesiredState
    Policy      ResourcePolicy
}
```

## 8. Ownership

```go
type Ownership string

const (
    Managed  Ownership = "managed"
    Adopted  Ownership = "adopted"
    External Ownership = "external"
)
```

### Managed

HWS создал или получил явное право управлять lifecycle ресурса.

### Adopted

Ресурс уже существовал; HWS может использовать/фокусировать его, но destructive operations требуют отдельной policy.

### External

HWS только наблюдает состояние.

Ownership относится к resolved runtime resource, а не всегда заранее известен в ResourceSpec.

## 9. ResourceIdentity

Нужен слой сопоставления domain resource и observed runtime object.

```go
type ResourceIdentity struct {
    ResourceID ResourceID
    InstanceID string
    Ownership  Ownership
    Evidence   IdentityEvidence
}
```

`InstanceID` может быть адаптер-специфичным, но не должен просачиваться в canonical config.

IdentityEvidence хранит достаточную информацию, чтобы объяснить, почему процесс/window был принят за требуемый ресурс.

## 10. DesiredWorkspaceState

```go
type DesiredWorkspaceState struct {
    WorkspaceID WorkspaceID
    DefinitionVersion string
    Revision string
    Resources []DesiredResource
    Layout LayoutIntent
}
```

Revision вычисляется детерминированно из canonical representation.

## 11. ObservedWorkspaceState

```go
type ObservedWorkspaceState struct {
    WorkspaceID WorkspaceID
    Revision string
    CapturedAt time.Time
    Resources []ObservedResource
    Layout ObservedLayout
    Capabilities CapabilitySnapshot
}
```

Observed state всегда имеет timestamp и source/capability context.

## 12. ResourceState

Не использовать один boolean `running` для всех ресурсов.

Начальная нормализованная модель:

```go
type Presence string

const (
    PresenceUnknown Presence = "unknown"
    PresenceAbsent  Presence = "absent"
    PresencePresent Presence = "present"
)

type Health string

const (
    HealthUnknown  Health = "unknown"
    HealthReady    Health = "ready"
    HealthDegraded Health = "degraded"
    HealthFailed   Health = "failed"
)
```

Специализированные adapters могут иметь richer state, но application layer получает нормализованную форму плюс typed details.

## 13. ReconcileDiff

```go
type ReconcileDiff struct {
    MissingRequired  []ResourceID
    MissingPreferred []ResourceID
    MissingOptional  []ResourceID
    Drifted          []ResourceID
    ExtraManaged     []ResourceID
    Unknown          []ResourceID
    LayoutDiff       LayoutDiff
}
```

Diff — pure value без side effects.

Его построение должно иметь property/unit tests.

## 14. ReconcilePlan

Diff преобразуется в набор intents.

```go
type IntentKind string

type Intent struct {
    ID         string
    Kind       IntentKind
    ResourceID ResourceID
    IdempotencyKey string
    DependencyIDs []string
}
```

ReconcilePlan не выполняет действия сам.

Axiom orchestration решает порядок/правила выполнения значимых intents.

## 15. Workspace lifecycle phase

Canonical application-level enum:

```go
type WorkspacePhase string

const (
    WorkspaceInactive   WorkspacePhase = "inactive"
    WorkspacePreparing  WorkspacePhase = "preparing"
    WorkspaceActive     WorkspacePhase = "active"
    WorkspaceDegraded   WorkspacePhase = "degraded"
    WorkspaceRecovering WorkspacePhase = "recovering"
    WorkspaceSuspending WorkspacePhase = "suspending"
    WorkspaceClosing    WorkspacePhase = "closing"
    WorkspaceFailed     WorkspacePhase = "failed"
)
```

Этот enum не должен автоматически совпадать с внутренними numeric/state IDs Axiom.

## 16. Operation

Пользовательский side-effect intent получает OperationID.

```go
type Operation struct {
    ID          OperationID
    WorkspaceID WorkspaceID
    Kind        OperationKind
    RequestedAt time.Time
    RequestedBy RequestSource
}
```

OperationID используется для correlation, но Axiom execution ID может иметь другую стратегию.

## 17. Capability

```go
type CapabilityID string

type CapabilityStatus string

const (
    CapabilityAvailable   CapabilityStatus = "available"
    CapabilityUnavailable CapabilityStatus = "unavailable"
    CapabilityUnknown     CapabilityStatus = "unknown"
)
```

Capability snapshot versioned и timestamped.

Нельзя интерпретировать `unknown` как `available`.

## 18. Actions

Action отличается от Workspace.

Workspace задаёт состояние, к которому HWS стремится.
Action — одно намеренное действие.

Примеры Action:

- Run tests;
- Open logs;
- Restart managed service;
- Connect remote;
- Capture diagnostics.

Action может использовать Axiom, если имеет значимые side effects/retry/recovery, но простая read-only action может выполняться application service без durable execution.

## 19. Error model

```go
type ErrorCode string

type HWSError struct {
    Code      ErrorCode
    Message   string
    Retryable bool
    ResourceID *ResourceID
    CauseClass string
}
```

Go error wrapping сохраняется внутри процесса, но IPC получает стабильное representation без сериализации произвольной error chain.

Начальные классы:

```text
validation.*
capability.*
resource.identity.*
activity.transient.*
activity.permanent.*
permission.*
storage.*
ipc.*
reconcile.*
recovery.*
```

## 20. Dynamic providers

Provider не возвращает произвольный UI-компонент. Он возвращает domain nodes/status data через versioned provider contract.

```go
type NodeProvider interface {
    Children(ctx context.Context, parent NodeID) (ProviderSnapshot, error)
}
```

Provider output проходит нормализацию/validation до публикации UI.

## 21. Pure core

Следующие функции должны быть максимально pure/deterministic:

- tree validation;
- path validation;
- canonical ordering;
- desired revision;
- diff desired/observed;
- reconcile intent generation;
- requirement evaluation;
- layout intent normalization.

Именно они являются главными кандидатами на property, fuzz и mutation testing.

## 22. Нерешённые вопросы

До M1 нужны отдельные решения:

1. формат stable IDs для user-authored config;
2. canonical serialization/hashing;
3. подробная layout model;
4. identity confidence/evidence model;
5. semantics `Suspend` vs `Close`;
6. динамические provider permissions;
7. migration strategy WorkspaceDefinition;
8. mapping Axiom lifecycle state ↔ application representation.
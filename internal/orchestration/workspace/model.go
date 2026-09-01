package workspace

import (
	"time"

	"github.com/Homiakus/axiom/model"
)

const (
	StatusInactive   = "inactive"
	StatusPreparing  = "preparing"
	StatusActive     = "active"
	StatusDegraded   = "degraded"
	StatusRecovering = "recovering"
	StatusClosing    = "closing"
	StatusFailed     = "failed"
)

type State struct {
	Status             string `json:"status"`
	WorkspaceID        string `json:"workspaceId"`
	DefinitionRevision string `json:"definitionRevision"`
	OperationKey       string `json:"operationKey"`
	ReachedRequired    int    `json:"reachedRequired"`
	TotalRequired      int    `json:"totalRequired"`
	LastFailureCode    string `json:"lastFailureCode"`
}

type Activate struct {
	WorkspaceID        string `json:"workspaceId"`
	DefinitionRevision string `json:"definitionRevision"`
	OperationKey       string `json:"operationKey"`
}

type Recover struct {
	OperationKey string `json:"operationKey"`
}

type Resume struct {
	OperationKey string `json:"operationKey"`
}

type Suspend struct{}

type Close struct {
	OperationKey string `json:"operationKey"`
}

type ReconcileWorkspaceInput struct {
	WorkspaceID        string `json:"workspaceId"`
	DefinitionRevision string `json:"definitionRevision"`
	OperationKey       string `json:"operationKey"`
}

type ReconcileWorkspaceOutput struct {
	Status          string `json:"status"`
	ReachedRequired int    `json:"reachedRequired"`
	TotalRequired   int    `json:"totalRequired"`
	FailureCode     string `json:"failureCode"`
}

type CloseWorkspaceInput struct {
	WorkspaceID        string `json:"workspaceId"`
	DefinitionRevision string `json:"definitionRevision"`
	OperationKey       string `json:"operationKey"`
}

type CloseWorkspaceOutput struct {
	Status      string `json:"status"`
	FailureCode string `json:"failureCode"`
}

var (
	stateStatus             = model.Key[State, string]("Status")
	stateWorkspaceID        = model.Key[State, string]("WorkspaceID")
	stateDefinitionRevision = model.Key[State, string]("DefinitionRevision")
	stateOperationKey       = model.Key[State, string]("OperationKey")
	stateReachedRequired    = model.Key[State, int]("ReachedRequired")
	stateTotalRequired      = model.Key[State, int]("TotalRequired")
	stateLastFailureCode    = model.Key[State, string]("LastFailureCode")

	activateWorkspaceID        = model.Key[Activate, string]("WorkspaceID")
	activateDefinitionRevision = model.Key[Activate, string]("DefinitionRevision")
	activateOperationKey       = model.Key[Activate, string]("OperationKey")
	recoverOperationKey        = model.Key[Recover, string]("OperationKey")
	resumeOperationKey         = model.Key[Resume, string]("OperationKey")
	closeOperationKey          = model.Key[Close, string]("OperationKey")
)

func BuildDefinition() *model.Definition {
	definition := model.New("HWSWorkspaceLifecycle").Version("1")
	state := model.Bind[State](definition, "Workspace")
	activate := model.EventOf[Activate](definition)
	recoverEvent := model.EventOf[Recover](definition)
	resumeEvent := model.EventOf[Resume](definition)
	suspendEvent := model.EventOf[Suspend](definition)
	closeEvent := model.EventOf[Close](definition)

	model.StateDefault(state, stateStatus, StatusInactive)
	model.StateDefault(state, stateWorkspaceID, "")
	model.StateDefault(state, stateDefinitionRevision, "")
	model.StateDefault(state, stateOperationKey, "")
	model.StateDefault(state, stateReachedRequired, 0)
	model.StateDefault(state, stateTotalRequired, 0)
	model.StateDefault(state, stateLastFailureCode, "")

	status := model.StateField(state, stateStatus)
	workspaceID := model.StateField(state, stateWorkspaceID)
	definitionRevision := model.StateField(state, stateDefinitionRevision)
	operationKey := model.StateField(state, stateOperationKey)
	reachedRequired := model.StateField(state, stateReachedRequired)
	totalRequired := model.StateField(state, stateTotalRequired)
	lastFailureCode := model.StateField(state, stateLastFailureCode)

	definition.Policy("workspaceIO").
		Retry(2).
		Timeout(5 * time.Second).
		Concurrency("once").
		Idempotency("required")

	definition.Activity("ReconcileWorkspace").
		Input("workspaceId", workspaceID).
		Input("definitionRevision", definitionRevision).
		Input("operationKey", operationKey).
		Output("status", "String").
		Output("reachedRequired", "Int").
		Output("totalRequired", "Int").
		Output("failureCode", "String").
		Effect("external").
		IdempotencyKey(operationKey).
		Policy("workspaceIO")

	definition.Activity("CloseWorkspace").
		Input("workspaceId", workspaceID).
		Input("definitionRevision", definitionRevision).
		Input("operationKey", operationKey).
		Output("status", "String").
		Output("failureCode", "String").
		Effect("external").
		IdempotencyKey(operationKey).
		Policy("workspaceIO")

	definition.Rule("activate").
		On(activate.Trigger()).
		Set(workspaceID, model.EventField(activate, activateWorkspaceID)).
		Set(definitionRevision, model.EventField(activate, activateDefinitionRevision)).
		Set(operationKey, model.EventField(activate, activateOperationKey)).
		Set(lastFailureCode, "").
		Set(status, StatusPreparing)

	definition.Rule("reconcilePreparing").
		On(model.StateChanged(state, stateStatus)).
		When(status.Equal(StatusPreparing)).
		Run("ReconcileWorkspace").
		Set(reachedRequired, model.OutputInt("reachedRequired")).
		Set(totalRequired, model.OutputInt("totalRequired")).
		Set(lastFailureCode, model.OutputString("failureCode")).
		Set(status, model.OutputString("status"))

	definition.Rule("recover").
		On(recoverEvent.Trigger()).
		Set(operationKey, model.EventField(recoverEvent, recoverOperationKey)).
		Set(status, StatusRecovering)

	definition.Rule("reconcileRecovering").
		On(model.StateChanged(state, stateStatus)).
		When(status.Equal(StatusRecovering)).
		Run("ReconcileWorkspace").
		Set(reachedRequired, model.OutputInt("reachedRequired")).
		Set(totalRequired, model.OutputInt("totalRequired")).
		Set(lastFailureCode, model.OutputString("failureCode")).
		Set(status, model.OutputString("status"))

	definition.Rule("resume").
		On(resumeEvent.Trigger()).
		Set(operationKey, model.EventField(resumeEvent, resumeOperationKey)).
		Set(status, StatusPreparing)

	definition.Rule("suspend").
		On(suspendEvent.Trigger()).
		Set(status, StatusInactive)

	definition.Rule("closeRequested").
		On(closeEvent.Trigger()).
		Set(operationKey, model.EventField(closeEvent, closeOperationKey)).
		Set(status, StatusClosing)

	definition.Rule("closeManagedResources").
		On(model.StateChanged(state, stateStatus)).
		When(status.Equal(StatusClosing)).
		Run("CloseWorkspace").
		Set(lastFailureCode, model.OutputString("failureCode")).
		Set(status, model.OutputString("status"))

	definition.Claim("requiredCountsNonNegative", reachedRequired.GreaterOrEqual(0))
	definition.Claim("totalRequiredNonNegative", totalRequired.GreaterOrEqual(0))
	definition.Claim("reachedDoesNotExceedTotal", reachedRequired.LessOrEqualField(totalRequired))
	definition.Claim(
		"activeRequiresWorkspaceIdentity",
		model.Implies(status.Equal(StatusActive), model.Exists(workspaceID.Expr())),
	)

	return definition
}

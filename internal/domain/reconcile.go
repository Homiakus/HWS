package domain

import "sort"

type ReconcileActionKind string

const (
	ActionEnsure ReconcileActionKind = "ensure"
)

type ReconcileAction struct {
	Kind     ReconcileActionKind
	Resource ResourceSpec
}

type ReconcilePlan struct {
	Actions            []ReconcileAction
	UnmanagedUnreached []ResourceID
}

func BuildReconcilePlan(desired DesiredState, observed ObservedState) ReconcilePlan {
	resources := append([]ResourceSpec(nil), desired.Resources...)
	sort.SliceStable(resources, func(i, j int) bool { return resources[i].ID < resources[j].ID })

	plan := ReconcilePlan{}
	for _, resource := range resources {
		observation, ok := observed.Resources[resource.ID]
		if ok && observation.Present && observation.Ready {
			continue
		}
		if resource.Ownership == OwnershipExternal {
			plan.UnmanagedUnreached = append(plan.UnmanagedUnreached, resource.ID)
			continue
		}
		plan.Actions = append(plan.Actions, ReconcileAction{Kind: ActionEnsure, Resource: resource})
	}
	return plan
}

type ReconcileEvaluation struct {
	RequiredTotal   int
	RequiredReached int
	OptionalTotal   int
	OptionalReached int
	MissingRequired []ResourceID
	MissingOptional []ResourceID
}

func EvaluateReconcile(desired DesiredState, observed ObservedState) ReconcileEvaluation {
	result := ReconcileEvaluation{}
	resources := append([]ResourceSpec(nil), desired.Resources...)
	sort.SliceStable(resources, func(i, j int) bool { return resources[i].ID < resources[j].ID })

	for _, resource := range resources {
		observation := observed.Resources[resource.ID]
		reached := observation.Present && observation.Ready
		if resource.Required {
			result.RequiredTotal++
			if reached {
				result.RequiredReached++
			} else {
				result.MissingRequired = append(result.MissingRequired, resource.ID)
			}
			continue
		}
		result.OptionalTotal++
		if reached {
			result.OptionalReached++
		} else {
			result.MissingOptional = append(result.MissingOptional, resource.ID)
		}
	}
	return result
}

package workspace

// isExecutionNotFound is intentionally isolated at the Axiom boundary.
// The pinned pre-v1 Axiom runtime currently exposes ErrExecutionNotFound only
// from an internal package, which Go correctly prevents HWS from importing.
// Keep this exact-message compatibility shim local and replace it with a
// public Axiom classifier as soon as that API exists.
func isExecutionNotFound(err error) bool {
	return err != nil && err.Error() == "execution not found"
}

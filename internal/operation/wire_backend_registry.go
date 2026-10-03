// WHY: This file preserves registration order while grouping backend operations by their public navigation and mutation phases.

package operation

import "fmt"

func registerNavigationOps(registry *Registry) error {
	if err := Register(registry, lookupDef()); err != nil {
		return fmt.Errorf("register lookup operation: %w", err)
	}
	if err := Register(registry, referencesDef()); err != nil {
		return fmt.Errorf("register references operation: %w", err)
	}
	return nil
}

func registerRenameOps(registry *Registry) error {
	if err := Register(registry, renameDef()); err != nil {
		return fmt.Errorf("register rename operation: %w", err)
	}
	return nil
}

func registerVerificationOps(registry *Registry) error {
	if err := Register(registry, verifyDef()); err != nil {
		return fmt.Errorf("register verify operation: %w", err)
	}
	return nil
}

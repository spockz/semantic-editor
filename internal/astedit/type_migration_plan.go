// WHY: Proves alias-equivalent type migrations before any source edits are applied.

package astedit

import (
	"fmt"
	"go/types"
)

// AliasMigrationProof records a type-name change whose types are identical after alias resolution.
type AliasMigrationProof struct {
	Current string
	Desired string
}

// PlanAliasTypeMigration accepts only types with exact identity after alias resolution.
func PlanAliasTypeMigration(currentName, desiredName string, current, desired types.Type) (AliasMigrationProof, error) {
	if currentName == "" || desiredName == "" || current == nil || desired == nil {
		return AliasMigrationProof{}, fmt.Errorf("type migration requires named source and destination types")
	}
	if !types.Identical(types.Unalias(current), types.Unalias(desired)) {
		return AliasMigrationProof{}, fmt.Errorf("type migration refused: %s and %s are not identical after alias resolution", currentName, desiredName)
	}
	return AliasMigrationProof{Current: currentName, Desired: desiredName}, nil
}

// WHY: This file registers the Go contract operations after their feature-specific handlers are defined.

package operation

import (
	"fmt"
)

func registerContractOps(registry *Registry) error {
	if err := Register(registry, extractInterfaceDef()); err != nil {
		return fmt.Errorf("register extract interface operation: %w", err)
	}
	if err := Register(registry, implementContractDef()); err != nil {
		return fmt.Errorf("register implement contract operation: %w", err)
	}
	return nil
}

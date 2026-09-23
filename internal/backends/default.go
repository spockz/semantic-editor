// Package backends assembles the built-in language implementations behind the neutral service.
package backends

import (
	"fmt"
	"semedit/internal/backend"
	javabackend "semedit/internal/backend/java"
)

// NewDefaultService assembles the built-in language implementations behind the shared service boundary.
func NewDefaultService() *backend.Service {
	registry, err := backend.NewRegistry(
		backend.NewGoBackend(),
		backend.NewRustBackend(),
		javabackend.NewJavaBackend(),
		backend.NewScalaBackend(),
		backend.NewHaskellBackend(),
	)
	if err != nil {
		panic(fmt.Sprintf("register built-in backends: %v", err))
	}
	return backend.NewService(registry)
}

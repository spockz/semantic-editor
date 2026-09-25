// Package backends assembles the built-in language implementations behind the neutral service.
package backends

import (
	"fmt"
	"semedit/internal/backend"
	gobackend "semedit/internal/backend/golang"
	haskellbackend "semedit/internal/backend/haskell"
	javabackend "semedit/internal/backend/java"
	kotlinbackend "semedit/internal/backend/kotlin"
	rustbackend "semedit/internal/backend/rust"
	scalabackend "semedit/internal/backend/scala"
)

// NewDefaultService assembles the built-in language implementations behind the shared service boundary.
func NewDefaultService() *backend.Service {
	registry, err := backend.NewRegistry(
		gobackend.NewGoBackend(),
		rustbackend.NewRustBackend(),
		javabackend.NewJavaBackend(),
		scalabackend.NewScalaBackend(),
		kotlinbackend.NewKotlinBackend(),
		haskellbackend.NewHaskellBackend(),
	)
	if err != nil {
		panic(fmt.Sprintf("register built-in backends: %v", err))
	}
	return backend.NewService(registry)
}

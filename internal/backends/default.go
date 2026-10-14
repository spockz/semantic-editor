// Package backends assembles the built-in language implementations behind the neutral service.
package backends

import (
	"fmt"
	"github.com/spockz/semantic-editor/internal/backend"
	bashbackend "github.com/spockz/semantic-editor/internal/backend/bash"
	gobackend "github.com/spockz/semantic-editor/internal/backend/golang"
	haskellbackend "github.com/spockz/semantic-editor/internal/backend/haskell"
	javabackend "github.com/spockz/semantic-editor/internal/backend/java"
	kotlinbackend "github.com/spockz/semantic-editor/internal/backend/kotlin"
	makebackend "github.com/spockz/semantic-editor/internal/backend/makefile"
	rustbackend "github.com/spockz/semantic-editor/internal/backend/rust"
	scalabackend "github.com/spockz/semantic-editor/internal/backend/scala"
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
		bashbackend.NewBackend(),
		makebackend.NewBackend(),
	)
	if err != nil {
		panic(fmt.Sprintf("register built-in backends: %v", err))
	}
	return backend.NewService(registry)
}

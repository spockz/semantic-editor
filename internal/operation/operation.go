// Package operation provides the central registry of semantic operations shared by CLI and MCP ingress.
package operation

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"semedit/internal/backend"
)

// Level identifies the scope an operation acts upon.
type Level string

const (
	// LevelSymbol marks operations resolving or editing a single symbol.
	LevelSymbol Level = "symbol"
	// LevelFile marks operations rewriting one file.
	LevelFile Level = "file"
	// LevelBuild marks operations formatting sources and reporting compiler diagnostics.
	LevelBuild Level = "build"
	// LevelWorkspace marks operations spanning the workspace.
	LevelWorkspace Level = "workspace"
)

// Parameter value types accepted by ParameterContract.
const (
	// ParamString accepts a JSON string or CLI string flag.
	ParamString = "string"
	// ParamBoolean accepts a JSON boolean or CLI boolean/string flag.
	ParamBoolean = "boolean"
	// ParamStringSlice accepts a JSON string array, a repeated flag, or a single string.
	ParamStringSlice = "string[]"
)

// ParameterContract declares one operation parameter in ingress-neutral form.
type ParameterContract struct {
	Name         string
	CLIName      string
	JSONName     string
	Type         string
	Description  string
	Required     bool
	Default      any
	Enums        []string
	DynamicEnums func(backend.Backend) []string
}

// Def declares one semantic operation with typed parse, per-language handlers, and formatting.
type Def[Req any, Res any] struct {
	Key          string
	Summary      string
	Params       []ParameterContract
	Level        Level
	CLIName      string
	MCPName      string
	PlacementKey bool
	Batchable    bool
	ReadOnly     bool
	Parse        func(map[string]any) (Req, error)
	Handlers     map[backend.LanguageID]func(context.Context, CallContext, Req) (Res, error)
	Format       func(Res) (string, error)
	ExampleRaw   map[string]any
}

// Entry is the type-erased view of a Def stored in a Registry.
type Entry struct {
	Key          string
	Summary      string
	Params       []ParameterContract
	Level        Level
	CLIName      string
	MCPName      string
	PlacementKey bool
	Batchable    bool
	ReadOnly     bool
	Languages    []backend.LanguageID
	ExampleRaw   map[string]any
	Parse        func(map[string]any) (any, error)
	Invoke       func(CallContext, map[string]any) (any, error)
	Format       func(any) (string, error)
}

// Registry stores operations indexed by key, MCP tool name, and CLI command name.
type Registry struct {
	byKey map[string]Entry
	byMCP map[string]Entry
	byCLI map[string]Entry
}

// ErrDuplicateOperation indicates an operation key, MCP name, or CLI name is already registered.
var ErrDuplicateOperation = errors.New("operation already registered")

// ErrInvalidDefinition indicates an operation definition is missing its key or parse function.
var ErrInvalidDefinition = errors.New("invalid operation definition")

// NewRegistry constructs an empty operation registry.
func NewRegistry() *Registry {
	return &Registry{
		byKey: make(map[string]Entry),
		byMCP: make(map[string]Entry),
		byCLI: make(map[string]Entry),
	}
}

// Register adds def to the registry, rejecting duplicate keys, MCP names, and CLI names.
// It is a function rather than a Registry method because Go methods cannot declare type parameters.
// Type erasure lives only in the Parse, Invoke, and Format closures built here.
func Register[Req any, Res any](registry *Registry, def Def[Req, Res]) error {
	if registry == nil {
		return fmt.Errorf("register %q: %w", def.Key, ErrInvalidDefinition)
	}
	if def.Key == "" || def.Parse == nil {
		return fmt.Errorf("register %q: %w", def.Key, ErrInvalidDefinition)
	}
	if _, exists := registry.byKey[def.Key]; exists {
		return fmt.Errorf("register %q: %w", def.Key, ErrDuplicateOperation)
	}
	if def.MCPName != "" {
		if _, exists := registry.byMCP[def.MCPName]; exists {
			return fmt.Errorf("register MCP tool %q: %w", def.MCPName, ErrDuplicateOperation)
		}
	}
	if def.CLIName != "" {
		if _, exists := registry.byCLI[def.CLIName]; exists {
			return fmt.Errorf("register CLI command %q: %w", def.CLIName, ErrDuplicateOperation)
		}
	}
	languages := make([]backend.LanguageID, 0, len(def.Handlers))
	for language := range def.Handlers {
		languages = append(languages, language)
	}
	slices.Sort(languages)
	entry := Entry{
		Key:          def.Key,
		Summary:      def.Summary,
		Params:       append([]ParameterContract(nil), def.Params...),
		Level:        def.Level,
		CLIName:      def.CLIName,
		MCPName:      def.MCPName,
		PlacementKey: def.PlacementKey,
		Batchable:    def.Batchable,
		ReadOnly:     def.ReadOnly,
		Languages:    languages,
		ExampleRaw:   cloneRaw(def.ExampleRaw),
		Parse: func(raw map[string]any) (any, error) {
			request, err := def.Parse(raw)
			if err != nil {
				return nil, err
			}
			return request, nil
		},
		Invoke: func(cc CallContext, raw map[string]any) (any, error) {
			request, err := def.Parse(raw)
			if err != nil {
				return nil, err
			}
			ctx := cc.Ctx
			if ctx == nil {
				ctx = context.Background()
			}
			project := effectiveProject(cc, projectOf(request))
			setProject(&request, project)
			// A lone LanguageAuto handler marks a language-independent operation:
			// invoke directly without backend language resolution.
			if len(def.Handlers) == 1 {
				if handler, ok := def.Handlers[backend.LanguageAuto]; ok && handler != nil {
					result, err := handler(ctx, cc, request)
					if err != nil {
						return nil, err
					}
					return result, nil
				}
			}
			language, err := resolveLanguage(cc, project)
			if err != nil {
				return nil, err
			}
			if handler, ok := def.Handlers[language]; ok && handler != nil {
				result, err := handler(ctx, cc, request)
				if err != nil {
					return nil, err
				}
				return result, nil
			}
			// LanguageAuto keys a language-independent handler used when resolution
			// yields no dedicated implementation (workspace-lifetime operations).
			if handler, ok := def.Handlers[backend.LanguageAuto]; ok && handler != nil {
				result, err := handler(ctx, cc, request)
				if err != nil {
					return nil, err
				}
				return result, nil
			}
			return nil, fmt.Errorf("dispatch %q for language %q: %w", def.Key, language, ErrUnsupportedLanguage)
		},
		Format: func(value any) (string, error) {
			result, ok := value.(Res)
			if !ok {
				return "", fmt.Errorf("format %q: unexpected result type %T: %w", def.Key, value, ErrInvalidDefinition)
			}
			if def.Format == nil {
				return fmt.Sprintf("%v", result), nil
			}
			return def.Format(result)
		},
	}
	registry.byKey[entry.Key] = entry
	if entry.MCPName != "" {
		registry.byMCP[entry.MCPName] = entry
	}
	if entry.CLIName != "" {
		registry.byCLI[entry.CLIName] = entry
	}
	return nil
}

// LookupKey returns the entry registered for an operation key.
func (r *Registry) LookupKey(key string) (Entry, bool) {
	if r == nil {
		return Entry{}, false
	}
	entry, ok := r.byKey[key]
	return entry, ok
}

// LookupMCP returns the entry registered for an MCP tool name.
func (r *Registry) LookupMCP(name string) (Entry, bool) {
	if r == nil {
		return Entry{}, false
	}
	entry, ok := r.byMCP[name]
	return entry, ok
}

// LookupCLI returns the entry registered for a CLI command name.
func (r *Registry) LookupCLI(name string) (Entry, bool) {
	if r == nil {
		return Entry{}, false
	}
	entry, ok := r.byCLI[name]
	return entry, ok
}

// All returns every registered entry sorted by key.
func (r *Registry) All() []Entry {
	if r == nil {
		return nil
	}
	return slices.SortedFunc(maps.Values(r.byKey), func(a, b Entry) int {
		return cmp.Compare(a.Key, b.Key)
	})
}

// ParametersForBackends builds parameter contracts using capabilities reported by
// the registered backend instances that can handle the operation.
func ParametersForBackends(entry Entry, backends []backend.Backend) []ParameterContract {
	params := append([]ParameterContract(nil), entry.Params...)
	for i := range params {
		if params[i].DynamicEnums == nil {
			continue
		}
		seen := map[string]bool{}
		var values []string
		for _, candidate := range backends {
			if !slices.Contains(entry.Languages, candidate.Language()) {
				continue
			}
			for _, value := range params[i].DynamicEnums(candidate) {
				if !seen[value] {
					seen[value] = true
					values = append(values, value)
				}
			}
		}
		params[i].Enums = values
	}
	return params
}

func cloneRaw(raw map[string]any) map[string]any {
	if raw == nil {
		return nil
	}
	cloned := make(map[string]any, len(raw))
	maps.Copy(cloned, raw)
	return cloned
}

func projectOf[Req any](request Req) backend.ProjectContext {
	if getter, ok := any(request).(interface{ GetProjectContext() backend.ProjectContext }); ok {
		return getter.GetProjectContext()
	}
	return backend.ProjectContext{}
}

func setProject[Req any](request *Req, project backend.ProjectContext) {
	if setter, ok := any(request).(interface{ SetProjectContext(backend.ProjectContext) }); ok {
		setter.SetProjectContext(project)
	}
}

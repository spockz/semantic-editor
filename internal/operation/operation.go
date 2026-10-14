// Package operation provides the central registry of semantic operations shared by CLI and MCP ingress.
package operation

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/capability"
)

// Entry is the type-erased view of an operation stored in the registry.
type Entry struct {
	Key                string
	VariantVerb        string
	VariantKind        string
	Summary            string
	Params             []ParameterContract
	Level              Level
	CLIName            string
	MCPName            string
	PlacementKey       bool
	Batchable          bool
	BufferedBatch      bool
	ReadOnly           bool
	Languages          []backend.LanguageID
	ExampleRaw         map[string]any
	NormalizeInput     func(map[string]any) (map[string]any, error)
	Output             *OperationOutputContract
	ErrorOutput        *OperationErrorOutputContract
	FinalizeOutput     func(CallContext, map[string]any, any) (any, error)
	ProjectOutput      func(any) (any, error)
	ProjectErrorOutput func(error) (value any, known bool, err error)
	Parse              func(map[string]any) (any, error)
	Invoke             func(CallContext, map[string]any) (any, error)
	Format             func(any) (string, error)
	LanguageContracts  map[backend.LanguageID]capability.OperationContract
}

// NormalizedInput returns canonical parameter spellings before a caller retains or forwards an operation request.
func (e Entry) NormalizedInput(raw map[string]any) (map[string]any, error) {
	return normalizedInput(e.NormalizeInput, raw, e.Params)
}

// SupportsBufferedBatch selects eligible Go and Java requests and preserves disk-only contracts.
func (e Entry) SupportsBufferedBatch(cc CallContext, raw map[string]any) (bool, error) {
	if !e.BufferedBatch {
		return false, nil
	}
	request, err := e.Parse(raw)
	if err != nil {
		return false, err
	}
	project := effectiveProject(cc, projectOf(request))
	language, err := resolveLanguage(cc, project)
	if err != nil {
		return false, err
	}
	if language != backend.LanguageGo && language != backend.LanguageJava {
		return false, nil
	}
	if req, ok := request.(RemoveReq); ok {
		return !req.SafeDelete, nil
	}
	return true, nil
}

// Def declares one semantic operation with typed parse, handlers, and formatting.
type Def[Req any, Res any] struct {
	Key           string
	VariantVerb   string
	VariantKind   string
	Summary       string
	Params        []ParameterContract
	Level         Level
	CLIName       string
	MCPName       string
	PlacementKey  bool
	Batchable     bool
	BufferedBatch bool
	ReadOnly      bool
	// NormalizeInput canonicalizes ingress spellings before parameter validation and parsing.
	NormalizeInput func(map[string]any) (map[string]any, error)
	// PrepareProject normalizes a concrete project after ambient context merges and before language dispatch.
	// Operations should set this only when a selected path must be classified from that effective root.
	PrepareProject func(backend.ProjectContext) (backend.ProjectContext, error)
	Parse          func(map[string]any) (Req, error)
	Handlers       map[backend.LanguageID]func(context.Context, CallContext, Req) (Res, error)
	Format         func(Res) (string, error)
	ExampleRaw     map[string]any
	Output         *OperationOutputContract
	ErrorOutput    *OperationErrorOutputContract
	// FinalizeOutput runs after the handler and mutation observer, before any ingress emits the result.
	FinalizeOutput func(CallContext, Req, Res) (Res, error)
	// ProjectOutput explicitly converts a native result into the declared JSON-facing result shape.
	ProjectOutput func(Res) (any, error)
	// ProjectErrorOutput projects known operation state carried by an error without changing dispatch error semantics.
	ProjectErrorOutput func(error) (value any, known bool, err error)
}

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
	// ParamObject accepts a closed JSON object validated by Schema.
	ParamObject = "object"
	// ParamArray accepts a typed JSON array validated by Schema.
	ParamArray = "array"
)

// Parameter value types accepted by ParameterContract.
const (
	// ParamString accepts a JSON string or CLI string flag.
	ParamString = "string"
	// ParamBoolean accepts a JSON boolean or CLI boolean/string flag.
	ParamBoolean = "boolean"
	// ParamStringSlice accepts a JSON string array, a repeated flag, or a single string.
	ParamStringSlice = "string[]"
	// ParamInteger accepts a JSON integer and an integer CLI flag.
	ParamInteger = "integer"
)

// ParameterContract declares one operation parameter in ingress-neutral form.
type ParameterContract struct {
	Name             string
	CLIName          string
	JSONName         string
	Type             string
	Description      string
	Required         bool
	Default          any
	Enums            []string
	DynamicEnums     func(backend.Backend) []string
	LanguageSelector bool
	InputAliases     []string
	// SourceFields names top-level structured output fields that can feed this parameter.
	SourceFields []string
	Schema       *ValueSchema
	Role         ParameterRole
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
	if err := validateRegistration(registry, def); err != nil {
		return err
	}
	entry := buildRegisteredEntry(def)
	publishRegisteredEntry(registry, entry)
	return nil
}

func validateDefOutput[Req any, Res any](def Def[Req, Res]) error {
	if err := ValidateOutputMode(def.Output, def.ProjectOutput != nil); err != nil {
		return fmt.Errorf("register %q: %w", def.Key, err)
	}
	if def.ProjectErrorOutput == nil {
		if def.ErrorOutput != nil {
			return fmt.Errorf("register %q error output schema requires an error projector: %w", def.Key, ErrInvalidDefinition)
		}
		return nil
	}
	if def.ErrorOutput == nil || def.ErrorOutput.ResultSchema == nil {
		return fmt.Errorf("register %q error projector requires an error output schema: %w", def.Key, ErrInvalidDefinition)
	}
	if err := validateErrorOutputSchema(def.ErrorOutput.ResultSchema); err != nil {
		return fmt.Errorf("register %q error output contract: %w", def.Key, err)
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
	eligible := make([]backend.Backend, 0, len(backends))
	for _, candidate := range backends {
		if slices.Contains(entry.Languages, candidate.Language()) {
			eligible = append(eligible, candidate)
		}
	}
	for i := range params {
		if params[i].LanguageSelector {
			seen := map[string]bool{}
			var values []string
			for _, candidate := range eligible {
				value := string(candidate.Language())
				if !seen[value] {
					seen[value] = true
					values = append(values, value)
				}
			}
			if slices.Contains(params[i].Enums, string(backend.LanguageAuto)) {
				values = append([]string{string(backend.LanguageAuto)}, values...)
			}
			params[i].Enums = values
		}
		if params[i].DynamicEnums == nil {
			continue
		}
		seen := map[string]bool{}
		var values []string
		for _, candidate := range eligible {
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

func invokeHandler[Req any, Res any](ctx context.Context, cc CallContext, request Req, handler func(context.Context, CallContext, Req) (Res, error), mutating bool) (result Res, resultErr error) {
	if mutating {
		if observer, ok := ctx.Value(mutationObserverContextKey{}).(func(context.Context) error); ok && observer != nil {
			defer func() { resultErr = errors.Join(resultErr, observer(ctx)) }()
		}
	}
	return handler(ctx, cc, request)
}

func finalizeOutputHook[Req any, Res any](def Def[Req, Res]) func(CallContext, map[string]any, any) (any, error) {
	if def.FinalizeOutput == nil {
		return nil
	}
	return func(cc CallContext, raw map[string]any, value any) (any, error) {
		normalized, err := normalizedInput(def.NormalizeInput, raw, def.Params)
		if err != nil {
			return nil, fmt.Errorf("normalize %q finalization input: %w", def.Key, err)
		}
		request, err := def.Parse(normalized)
		if err != nil {
			return nil, fmt.Errorf("finalize %q request: %w", def.Key, err)
		}
		result, ok := value.(Res)
		if !ok {
			return nil, fmt.Errorf("finalize %q: unexpected result type %T: %w", def.Key, value, ErrInvalidDefinition)
		}
		finalized, err := def.FinalizeOutput(cc, request, result)
		if err != nil {
			return nil, fmt.Errorf("finalize %q output: %w", def.Key, err)
		}
		return finalized, nil
	}
}

func projectOutputHook[Req any, Res any](def Def[Req, Res]) func(any) (any, error) {
	if def.ProjectOutput == nil {
		return nil
	}
	return func(value any) (any, error) {
		result, ok := value.(Res)
		if !ok {
			return nil, fmt.Errorf("project %q: unexpected result type %T: %w", def.Key, value, ErrInvalidDefinition)
		}
		projected, err := def.ProjectOutput(result)
		if err != nil {
			return nil, fmt.Errorf("project %q result: %w", def.Key, err)
		}
		return normalizeAndValidateOutput(projected, def.Output)
	}
}

func projectErrorOutputHook[Req any, Res any](def Def[Req, Res]) func(error) (any, bool, error) {
	if def.ProjectErrorOutput == nil {
		return nil
	}
	return func(err error) (any, bool, error) {
		projected, known, projectErr := def.ProjectErrorOutput(err)
		if projectErr != nil {
			return nil, false, fmt.Errorf("project %q error result: %w", def.Key, projectErr)
		}
		if !known {
			return nil, false, nil
		}
		validated, validationErr := normalizeAndValidateOutput(projected, &OperationOutputContract{ResultSchema: def.ErrorOutput.ResultSchema})
		if validationErr != nil {
			return nil, false, fmt.Errorf("validate %q error result: %w", def.Key, validationErr)
		}
		if liveFieldErr := rejectLiveErrorFields(validated, "result"); liveFieldErr != nil {
			return nil, false, fmt.Errorf("validate %q error result: %w", def.Key, liveFieldErr)
		}
		return validated, true, nil
	}
}

// ValidateOutputMode enforces that every declared output contract specifies both a typed result schema and an output projector.
func ValidateOutputMode(contract *OperationOutputContract, hasProjector bool) error {
	if contract == nil {
		if hasProjector {
			return fmt.Errorf("output projector requires an output contract: %w", ErrInvalidDefinition)
		}
		return nil
	}
	if contract.ResultSchema == nil {
		return fmt.Errorf("output contract requires a typed result schema: %w", ErrInvalidDefinition)
	}
	if !hasProjector {
		return fmt.Errorf("typed output requires a projector: %w", ErrInvalidDefinition)
	}
	if err := ValidateOutputContract(*contract); err != nil {
		return fmt.Errorf("output contract: %w", err)
	}
	return nil
}

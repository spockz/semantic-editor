// Package operation adapts typed operation definitions into safe type-erased registry entries.
package operation

import (
	"context"
	"fmt"
	"slices"

	"semedit/internal/backend"
	"semedit/internal/capability"
)

func validateRegistration[Req any, Res any](registry *Registry, def Def[Req, Res]) error {
	if err := ValidateParameterContracts(def.Params); err != nil {
		return fmt.Errorf("register %q parameters: %w", def.Key, err)
	}
	if err := validateDefOutput(def); err != nil {
		return err
	}
	if registry == nil || def.Key == "" || def.Parse == nil {
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
	return nil
}

func buildRegisteredEntry[Req any, Res any](def Def[Req, Res]) Entry {
	languages := make([]backend.LanguageID, 0, len(def.Handlers))
	for language := range def.Handlers {
		languages = append(languages, language)
	}
	slices.Sort(languages)
	entry := Entry{
		Key:            def.Key,
		VariantVerb:    def.VariantVerb,
		VariantKind:    def.VariantKind,
		Summary:        def.Summary,
		Params:         append([]ParameterContract(nil), def.Params...),
		Level:          def.Level,
		CLIName:        def.CLIName,
		MCPName:        def.MCPName,
		PlacementKey:   def.PlacementKey,
		Batchable:      def.Batchable,
		BufferedBatch:  def.BufferedBatch,
		ReadOnly:       def.ReadOnly,
		Languages:      languages,
		ExampleRaw:     cloneRaw(def.ExampleRaw),
		NormalizeInput: def.NormalizeInput,
		Output:         def.Output,
		ErrorOutput:    def.ErrorOutput,
		Parse:          registeredParser(def),
		Invoke:         registeredInvoker(def),
		Format:         registeredFormatter(def),
	}
	entry.FinalizeOutput = finalizeOutputHook(def)
	entry.ProjectOutput = projectOutputHook(def)
	entry.ProjectErrorOutput = projectErrorOutputHook(def)
	return entry
}

func publishRegisteredEntry(registry *Registry, entry Entry) {
	registry.byKey[entry.Key] = entry
	if entry.MCPName != "" {
		registry.byMCP[entry.MCPName] = entry
	}
	if entry.CLIName != "" {
		registry.byCLI[entry.CLIName] = entry
	}
}

func registeredParser[Req any, Res any](def Def[Req, Res]) func(map[string]any) (any, error) {
	return func(raw map[string]any) (any, error) {
		normalized, err := normalizedInput(def.NormalizeInput, raw)
		if err != nil {
			return nil, fmt.Errorf("normalize %q input: %w", def.Key, err)
		}
		if err = ValidateParameterValues(normalized, def.Params); err != nil {
			return nil, err
		}
		request, err := def.Parse(normalized)
		if err != nil {
			return nil, err
		}
		return request, nil
	}
}

func registeredInvoker[Req any, Res any](def Def[Req, Res]) func(CallContext, map[string]any) (any, error) {
	return func(cc CallContext, raw map[string]any) (any, error) {
		return invokeRegistered(def, cc, raw)
	}
}

func invokeRegistered[Req any, Res any](def Def[Req, Res], cc CallContext, raw map[string]any) (any, error) {
	normalized, err := normalizedInput(def.NormalizeInput, raw)
	if err != nil {
		return nil, fmt.Errorf("normalize %q input: %w", def.Key, err)
	}
	if err = ValidateParameterValues(normalized, def.Params); err != nil {
		return nil, err
	}
	request, err := def.Parse(normalized)
	if err != nil {
		return nil, err
	}
	ctx := cc.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	project := effectiveProject(cc, projectOf(request))
	if def.PrepareProject != nil {
		project, err = def.PrepareProject(project)
		if err != nil {
			return nil, fmt.Errorf("prepare %q project: %w", def.Key, err)
		}
	}
	setProject(&request, project)
	return dispatchRegisteredHandler(ctx, def, cc, request, project)
}

func dispatchRegisteredHandler[Req any, Res any](ctx context.Context, def Def[Req, Res], cc CallContext, request Req, project backend.ProjectContext) (any, error) {
	if len(def.Handlers) == 1 {
		if result, handled, err := invokeLanguageAuto(ctx, def, cc, request); handled || err != nil {
			return result, err
		}
	}
	language, err := resolveLanguage(cc, project)
	if err != nil {
		if def.Key == capability.OpVerify {
			if result, handled, invokeErr := invokeLanguageAuto(ctx, def, cc, request); handled || invokeErr != nil {
				return result, invokeErr
			}
		}
		return nil, err
	}
	if handler := def.Handlers[language]; handler != nil {
		return invokeHandler(ctx, cc, request, handler, !def.ReadOnly)
	}
	if result, handled, err := invokeLanguageAuto(ctx, def, cc, request); handled || err != nil {
		return result, err
	}
	return nil, fmt.Errorf("dispatch %q for language %q: %w", def.Key, language, ErrUnsupportedLanguage)
}

func invokeLanguageAuto[Req any, Res any](ctx context.Context, def Def[Req, Res], cc CallContext, request Req) (any, bool, error) {
	handler := def.Handlers[backend.LanguageAuto]
	if handler == nil {
		return nil, false, nil
	}
	result, err := invokeHandler(ctx, cc, request, handler, !def.ReadOnly)
	return result, true, err
}

func registeredFormatter[Req any, Res any](def Def[Req, Res]) func(any) (string, error) {
	return func(value any) (string, error) {
		result, ok := value.(Res)
		if !ok {
			return "", fmt.Errorf("format %q: unexpected result type %T: %w", def.Key, value, ErrInvalidDefinition)
		}
		if def.Format == nil {
			return fmt.Sprintf("%v", result), nil
		}
		return def.Format(result)
	}
}

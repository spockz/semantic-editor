// Package operation_test verifies backend-wired operation contracts without invoking language backends.
package operation_test

import (
	"context"
	"errors"
	"go/parser"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	gobackend "github.com/spockz/semantic-editor/internal/backend/golang"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/operation"
)

func TestMoveConstructDefaultsDestinationToSourceFile(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupCLI("move-construct")
	if !ok {
		t.Fatal("move_construct registry entry missing")
	}
	request, err := entry.Parse(map[string]any{"file": "pkg/source.go", "symbol": "MoveMe"})
	if err != nil {
		t.Fatal(err)
	}
	move, ok := request.(operation.MoveConstructReq)
	if !ok || move.DestinationFile != "pkg/source.go" {
		t.Fatalf("parsed move request = %#v, want destination defaulted to source file", request)
	}
}

func TestMoveConstructRejectsTargetWithoutRelativePlacement(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupCLI("move-construct")
	if !ok {
		t.Fatal("move_construct registry entry missing")
	}
	for name, raw := range map[string]map[string]any{
		"default placement": {"file": "pkg/source.go", "symbol": "MoveMe", "target_symbol": "Anchor"},
		"file placement":    {"file": "pkg/source.go", "symbol": "MoveMe", "placement": "file_start", "target_symbol": "Anchor"},
		"section placement": {"file": "pkg/source.go", "symbol": "MoveMe", "placement": "public_start", "target_symbol": "Anchor"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := entry.Parse(raw); !errors.Is(err, operation.ErrInvalidParams) {
				t.Fatalf("Parse(%#v) error = %v, want ErrInvalidParams", raw, err)
			}
		})
	}
	if _, err := entry.Parse(map[string]any{"file": "pkg/source.go", "symbol": "MoveMe", "placement": "before_symbol", "target_symbol": "Anchor"}); err != nil {
		t.Fatalf("relative target parse failed: %v", err)
	}
}

func TestJavaVerifyRegistryRequestAndDispatch(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupCLI("verify")
	if !ok {
		t.Fatal("verify registry entry missing")
	}
	request, err := entry.Parse(map[string]any{"file": "src/Widget.java", "language": "java", "trust_workspace": true, "jdtls_home": "/jdtls", "java_bin": "/java", "import_maven": true, "format_selected_file": true, "organize_imports": true})
	if err != nil {
		t.Fatal(err)
	}
	verify, ok := request.(operation.VerifyReq)
	if !ok || verify.Path != "src/Widget.java" || !verify.FormatSelectedFile || !verify.OrganizeImports || !verify.Project.Java.ImportMaven {
		t.Fatalf("parsed Java verify request = %#v", request)
	}
}

func TestAutomaticMutationDescriptionsDiscourageRedundantVerification(t *testing.T) {
	t.Parallel()

	registry := operation.DefaultRegistry()
	for _, name := range []string{
		"semantic_rename",
		"semantic_insert_declaration",
		"semantic_insert_function",
		"semantic_insert_type",
		"semantic_insert_decl",
		"semantic_insert_construct",
		"semantic_organize_imports",
		"semantic_replace_body",
		"semantic_extract_function",
		"semantic_extract_variable",
		"semantic_extract_constant",
	} {
		entry, ok := registry.LookupMCP(name)
		if !ok {
			t.Errorf("missing %s", name)
			continue
		}
		if !strings.Contains(entry.Summary, "do not call semantic_verify separately") {
			t.Errorf("%s does not discourage redundant verification: %q", name, entry.Summary)
		}
	}
}

func cloneExample(raw map[string]any) map[string]any {
	cloned := make(map[string]any, len(raw))
	maps.Copy(cloned, raw)
	return cloned
}

type verifyPolicyGoBackend struct {
	gobackend.GoBackend

	calls int
}

func (b *verifyPolicyGoBackend) Verify(context.Context, backend.VerifyRequest) ([]backend.Diagnostic, error) {
	b.calls++
	return nil, nil
}

func TestRegisteredDefsHonorContracts(t *testing.T) {
	t.Parallel()

	registry := operation.DefaultRegistry()
	entries := registry.All()
	if len(entries) != 46 {
		t.Errorf("registered operations = %d, want 46", len(entries))
	}
	for _, entry := range entries {
		t.Run(entry.Key, func(t *testing.T) {
			t.Parallel()
			assertRegisteredEntryContract(t, entry)
		})
	}
}

func assertRegisteredEntryContract(t *testing.T, entry operation.Entry) {
	t.Helper()
	missingParams := len(entry.Params) == 0 && !zeroArgumentStageControl(entry.Key)
	if entry.Key == "" || entry.Summary == "" || entry.MCPName == "" || missingParams {
		t.Errorf("entry has incomplete public metadata: %+v", entry)
	}
	if _, err := entry.Parse(entry.ExampleRaw); err != nil {
		t.Errorf("ExampleRaw parse failed: %v", err)
	}
	if zeroArgumentStageControl(entry.Key) {
		if _, err := entry.Parse(map[string]any{"unexpected": true}); err == nil {
			t.Error("zero-argument stage control accepted an unknown parameter")
		}
	}
	t.Run("cli-string-shapes", func(t *testing.T) {
		t.Parallel()
		assertBooleanCLIStringsParse(t, entry)
	})
	assertRequiredEntryParamsFail(t, entry)
	assertEnumEntryParamsFail(t, entry)
}

func zeroArgumentStageControl(key string) bool {
	return key == "stage_start" || key == "stage_abort"
}

func assertBooleanCLIStringsParse(t *testing.T, entry operation.Entry) {
	t.Helper()
	example := stringBooleanExample(entry)
	if !example.Converted {
		t.Skip("no boolean parameter in ExampleRaw")
	}
	if _, err := entry.Parse(example.Raw); err != nil {
		t.Errorf("CLI string boolean parse failed: %v", err)
	}
}

type booleanCLIExample struct {
	Raw       map[string]any
	Converted bool
}

func stringBooleanExample(entry operation.Entry) booleanCLIExample {
	shaped := cloneExample(entry.ExampleRaw)
	example := booleanCLIExample{Raw: shaped}
	for _, param := range entry.Params {
		if param.Type != operation.ParamBoolean {
			continue
		}
		value, exists := shaped[param.JSONName]
		if !exists {
			continue
		}
		flag, isBool := value.(bool)
		if !isBool {
			continue
		}
		shaped[param.JSONName] = strconv.FormatBool(flag)
		example.Converted = true
	}
	return example
}

func assertRequiredEntryParamsFail(t *testing.T, entry operation.Entry) {
	t.Helper()
	for _, param := range entry.Params {
		if !param.Required {
			continue
		}

		t.Run("missing-required-"+param.JSONName, func(t *testing.T) {
			t.Parallel()
			without := cloneExample(entry.ExampleRaw)
			delete(without, param.JSONName)
			delete(without, param.CLIName)
			if _, err := entry.Parse(without); err == nil {
				t.Errorf("parse without required param %q succeeded, want error", param.JSONName)
			} else if !errors.Is(err, operation.ErrInvalidParams) {
				t.Errorf("missing required param error = %v, want ErrInvalidParams", err)
			}
		})
	}
}

func assertEnumEntryParamsFail(t *testing.T, entry operation.Entry) {
	t.Helper()
	for _, param := range entry.Params {
		if len(param.Enums) == 0 {
			continue
		}

		t.Run("bad-enum-"+param.JSONName, func(t *testing.T) {
			t.Parallel()
			invalid := cloneExample(entry.ExampleRaw)
			invalid[param.JSONName] = "___not_a_value___"
			if _, err := entry.Parse(invalid); err == nil {
				t.Errorf("parse with bad enum %q succeeded, want error", param.JSONName)
			} else if !errors.Is(err, operation.ErrInvalidParams) {
				t.Errorf("bad enum error = %v, want ErrInvalidParams", err)
			}
		})
	}
}

func TestOperationExamplesUseSupportedJSONParameters(t *testing.T) {
	t.Parallel()

	for _, entry := range operation.DefaultRegistry().All() {
		if entry.ExampleRaw == nil {
			continue
		}
		t.Run(entry.Key, func(t *testing.T) {
			t.Parallel()

			supported := make(map[string]struct{}, len(entry.Params))
			for _, param := range entry.Params {
				supported[param.JSONName] = struct{}{}
			}
			for key := range entry.ExampleRaw {
				if _, ok := supported[key]; !ok {
					t.Errorf("ExampleRaw key %q is not a supported JSON parameter", key)
				}
			}
			if _, err := entry.Parse(entry.ExampleRaw); err != nil {
				t.Errorf("ExampleRaw violates the operation parameter contract: %v", err)
			}
		})
	}
}

func TestChangeSignatureExampleArgumentParsesAsGoExpression(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupCLI("change-signature")
	if !ok {
		t.Fatal("change-signature registry entry missing")
	}
	parameters, ok := entry.ExampleRaw["parameters"].([]any)
	if !ok || len(parameters) != 2 {
		t.Fatalf("change-signature example parameters = %#v", entry.ExampleRaw["parameters"])
	}
	added, ok := parameters[1].(map[string]any)
	if !ok {
		t.Fatalf("appended example mapping = %#v", parameters[1])
	}
	argument, ok := added["argument"].(string)
	if !ok {
		t.Fatalf("appended example argument = %#v", added["argument"])
	}
	if _, err := parser.ParseExpr(argument); err != nil {
		t.Fatalf("change-signature example argument %q is not a Go expression: %v", argument, err)
	}
}

func TestRegistryLookupsCoverWiredOperations(t *testing.T) {
	t.Parallel()

	registry := operation.DefaultRegistry()
	if _, ok := registry.LookupKey("lookup"); !ok {
		t.Error("LookupKey(lookup) missed")
	}
	if _, ok := registry.LookupMCP("semantic_lookup"); !ok {
		t.Error("LookupMCP(semantic_lookup) missed")
	}
	if _, ok := registry.LookupMCP("semantic_rename"); !ok {
		t.Error("LookupMCP(semantic_rename) missed")
	}
	if _, ok := registry.LookupMCP("semantic_verify"); !ok {
		t.Error("LookupMCP(semantic_verify) missed")
	}
	if _, ok := registry.LookupMCP("semantic_inspect_symbol"); !ok {
		t.Error("LookupMCP(semantic_inspect_symbol) missed")
	}
	if _, ok := registry.LookupMCP("semantic_outline"); !ok {
		t.Error("LookupMCP(semantic_outline) missed")
	}
	if _, ok := registry.LookupCLI("lookup"); !ok {
		t.Error("LookupCLI(lookup) missed")
	}
	if _, ok := registry.LookupCLI("rename"); !ok {
		t.Error("LookupCLI(rename) missed")
	}
	if _, ok := registry.LookupCLI("verify"); !ok {
		t.Error("LookupCLI(verify) missed")
	}
	if _, ok := registry.LookupCLI("inspect-symbol"); !ok {
		t.Error("LookupCLI(inspect-symbol) missed")
	}
	if _, ok := registry.LookupCLI("outline"); !ok {
		t.Error("LookupCLI(outline) missed")
	}
}

func TestMatrixDerivesSupportFromHandlers(t *testing.T) {
	t.Parallel()
	registry := operation.DefaultRegistry()
	matrix, err := registry.Matrix(backend.LanguageGo)
	if err != nil {
		t.Errorf("Matrix(go) failed: %v", err)
	}
	for _, key := range []string{"lookup", "rename", "verify", "find_references"} {
		capability, ok := matrix.Operations[key]
		if !ok {
			t.Errorf("go matrix misses operation %q", key)
			continue
		}
		if !capability.Supported {
			t.Errorf("go matrix marks %q unsupported", key)
		}
	}
	scala, err := registry.Matrix(backend.LanguageScala)
	if err != nil {
		t.Errorf("Matrix(scala) failed: %v", err)
	}
	if !scala.Operations["lookup"].Supported {
		t.Error("scala matrix marks lookup unsupported")
	}
	if scala.Operations["rename"].Supported {
		t.Error("scala matrix marks rename supported")
	}
	if scala.Operations["verify"].Supported {
		t.Error("scala matrix marks verify supported")
	}
	if _, err := registry.Matrix("cobol"); !errors.Is(err, operation.ErrUnsupportedLanguage) {
		t.Errorf("Matrix(cobol) error = %v, want ErrUnsupportedLanguage", err)
	}
}

func TestReadOperationsAreOnlyReadOnlyOperations(t *testing.T) {
	t.Parallel()

	for _, entry := range operation.DefaultRegistry().All() {
		want := entry.Key == "lookup" || entry.Key == "inspect_symbol" || entry.Key == "inspect_construct" || entry.Key == "outline" || entry.Key == "find_references" || entry.Key == "workspace_outline"
		if entry.ReadOnly != want {
			t.Errorf("operation %q ReadOnly = %v, want %v", entry.Key, entry.ReadOnly, want)
		}
		if (entry.Key == "find_references" || entry.Key == "workspace_outline") && entry.Batchable {
			t.Errorf("%s must not be batchable", entry.Key)
		}
	}
}

func TestLookupSharedProjectParserPreservesBackendConfiguration(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupCLI("lookup")
	if !ok {
		t.Fatal("lookup registry entry missing")
	}
	request, err := entry.Parse(map[string]any{
		"symbol":             "Widget",
		"file":               "src/Widget.java",
		"language":           "java",
		"trust_workspace":    true,
		"jdtls_home":         "/jdtls",
		"java_bin":           "/java",
		"import_maven":       true,
		"metals_home":        "/metals",
		"metals_bin":         "/metals/bin",
		"java_version":       "21",
		"standalone_haskell": true,
		"ghc_bin":            "/ghc",
		"hls_bin":            "/hls",
		"ghc_version":        "9.10",
		"hls_version":        "2.11",
		"kotlin_bin":         "/kotlin",
		"bash_bin":           "/bash",
		"make_bin":           "/make",
	})
	if err != nil {
		t.Fatal(err)
	}
	lookup, ok := request.(operation.LookupReq)
	if !ok {
		t.Fatalf("parsed request type = %T, want LookupReq", request)
	}
	want := backend.ProjectContext{
		File:           "src/Widget.java",
		Language:       backend.LanguageJava,
		WorkspaceTrust: backend.WorkspaceTrust{Trusted: true},
		Java:           backend.JavaConfig{JDTLSHome: "/jdtls", JavaBin: "/java", ImportMaven: true, ImportMavenSet: true},
		Scala:          backend.ScalaConfig{MetalsHome: "/metals", MetalsBin: "/metals/bin", JavaBin: "/java", JavaVersion: "21"},
		Haskell:        backend.HaskellConfig{Standalone: true, GHCBin: "/ghc", HLSBin: "/hls", GHCVersion: "9.10", HLSVersion: "2.11"},
		Kotlin:         backend.KotlinConfig{KotlinBin: "/kotlin"},
		Bash:           backend.BashConfig{BashBin: "/bash"},
		Make:           backend.MakeConfig{MakeBin: "/make"},
	}
	if !reflect.DeepEqual(lookup.Project, want) {
		t.Errorf("lookup project = %#v, want %#v", lookup.Project, want)
	}
	if lookup.Symbol != "Widget" {
		t.Errorf("lookup symbol = %q, want Widget", lookup.Symbol)
	}
}

func TestOutlineDirectoryDispatchUsesEffectiveRootBeforeLanguageSelection(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/outline\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "generated.java")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "source.go"), []byte("package sample\nfunc Ready() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, explicitLanguage := range []bool{false, true} {
		name := "auto language"
		raw := map[string]any{"path": "generated.java"}
		if explicitLanguage {
			name = "explicit language"
			raw["language"] = "go"
		}
		t.Run(name, func(t *testing.T) {
			cc := operation.NewCallContext(root, backend.ProjectContext{RootDir: root})
			result, err := cc.Registry.Dispatch(cc, "outline", raw)
			if err != nil {
				t.Fatal(err)
			}
			outline, ok := result.(*backend.OutlineResult)
			if !ok {
				t.Fatalf("result type = %T, want *backend.OutlineResult", result)
			}
			if outline.Scope.Path != "generated.java" || len(outline.Files) != 1 || outline.Files[0].File != "generated.java/source.go" {
				t.Fatalf("outline = %#v, want selected directory contents", outline)
			}
		})
	}
}

func TestFindReferencesRegistryAliases(t *testing.T) {
	registry := operation.DefaultRegistry()
	if _, ok := registry.LookupKey("find_references"); !ok {
		t.Fatal("LookupKey(find_references) missed")
	}
	if _, ok := registry.LookupMCP("semantic_find_references"); !ok {
		t.Fatal("LookupMCP(semantic_find_references) missed")
	}
	if _, ok := registry.LookupCLI("find-references"); !ok {
		t.Fatal("LookupCLI(find-references) missed")
	}
}

func TestOperationParamsRejectUnknownKeysAndAcceptCLIAliases(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupMCP("semantic_verify")
	if !ok {
		t.Fatal("verify registry entry missing")
	}
	if _, err := entry.Parse(map[string]any{"files": []string{"src/Widget.java"}}); err == nil || !strings.Contains(err.Error(), "supported keys") || !strings.Contains(err.Error(), "file") {
		t.Fatalf("unknown verify parameter error = %v, want supported-key list", err)
	}
	request, err := entry.Parse(map[string]any{
		"file":            "src/Widget.java",
		"language":        "java",
		"trust-workspace": true,
		"jdtls-home":      "/jdtls",
		"java-bin":        "/java",
	})
	if err != nil {
		t.Fatalf("valid CLI aliases rejected: %v", err)
	}
	verify, ok := request.(operation.VerifyReq)
	if !ok || !verify.Project.WorkspaceTrust.Trusted || verify.Project.Java.JDTLSHome != "/jdtls" || verify.Project.Java.JavaBin != "/java" {
		t.Fatalf("parsed verify request = %#v", request)
	}
}

func TestRemoveConstructSelectorContract(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupCLI("remove-construct")
	if !ok {
		t.Fatal("remove-construct registry entry missing")
	}
	request, err := entry.Parse(map[string]any{"file": "api/routes.go", "kind": "case", "in_function": "Route", "discriminator": "route", "target_case": `"old"`})
	if err != nil {
		t.Fatal(err)
	}
	remove, ok := request.(operation.RemoveReq)
	if !ok || remove.Symbol != "" || remove.InFunction != "Route" || remove.TargetCase != `"old"` || remove.SafeDelete {
		t.Fatalf("parsed remove request = %#v", request)
	}
	if _, err := entry.Parse(map[string]any{"file": "api/routes.go", "kind": "case", "symbol": "Route", "in_function": "Route"}); err == nil || !strings.Contains(err.Error(), "symbol is a declaration target") {
		t.Fatalf("case selection with symbol should fail: %v", err)
	}
}

func TestRenameProjectionUsesOneVerifiedPostRenameIdentity(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupMCP("semantic_rename")
	if !ok {
		t.Fatal("rename operation missing")
	}
	root := t.TempDir()
	location := backend.SourceLocation{URI: pathutil.FileURI(filepath.Join(root, "config.go")), Range: backend.Range{Start: backend.Position{Line: 2, Character: 4}}}
	result := &backend.RenameResult{
		ApplicationState: "persisted",
		ProjectRoot:      root,
		Source:           &backend.LookupResult{Definition: &backend.NamedTarget{Name: "ListenPort", QualifiedName: "Config.ListenPort", Location: location}},
		Lookup:           &backend.LookupResult{Definition: &backend.NamedTarget{Name: "BoundPort", QualifiedName: "Config.BoundPort", Location: location}},
		Revision:         "post-rename-digest",
	}
	projected, err := entry.ProjectOutput(result)
	if err != nil {
		t.Fatalf("project verified rename: %v", err)
	}
	identity := projected.(map[string]any)
	if identity["file"] != "config.go" || identity["symbol"] != "Config.BoundPort" || identity["expected_revision"] != "post-rename-digest" {
		t.Fatalf("verified aliases do not describe one post-rename identity: %#v", identity)
	}
	if origin := identity["origin"].(map[string]any); origin["symbol"] != "Config.ListenPort" {
		t.Fatalf("historical origin identity = %#v", origin)
	}
	result.Revision = ""
	projected, err = entry.ProjectOutput(result)
	if err != nil {
		t.Fatalf("project rename without final digest: %v", err)
	}
	identity = projected.(map[string]any)
	for _, field := range []string{"file", "symbol", "expected_revision"} {
		if _, exists := identity[field]; exists {
			t.Errorf("unverified final identity advertised %q: %#v", field, identity)
		}
	}
}

func TestRenameDiagnosticErrorProjectionHasNoLiveIdentity(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupMCP("semantic_rename")
	if !ok {
		t.Fatal("rename operation missing")
	}
	root := t.TempDir()
	result := &backend.RenameResult{
		ApplicationState: "persisted",
		ProjectRoot:      root,
		Source:           &backend.LookupResult{Definition: &backend.NamedTarget{Name: "ListenPort", QualifiedName: "Config.ListenPort", Location: backend.SourceLocation{URI: pathutil.FileURI(filepath.Join(root, "config.go"))}}},
	}
	projected, known, err := entry.ProjectErrorOutput(&backend.RenameDiagnosticsError{Result: result, IntroducedErrors: []string{"new error"}})
	if err != nil || !known {
		t.Fatalf("project applied rename diagnostic failure: known=%v err=%v", known, err)
	}
	output := projected.(map[string]any)
	if output["status"] != "applied_with_diagnostics" || output["application_state"] != "persisted" {
		t.Fatalf("rename error state = %#v", output)
	}
	for _, forbidden := range []string{"file", "symbol", "expected_revision", "continuations"} {
		if _, exists := output[forbidden]; exists {
			t.Errorf("rename error advertised live field %q: %#v", forbidden, output)
		}
	}
}

func TestRenameProjectionIncludesPublicTypeTargetFile(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupMCP("semantic_rename")
	if !ok {
		t.Fatal("rename operation missing")
	}
	root := t.TempDir()
	projected, err := entry.ProjectOutput(&backend.RenameResult{
		ApplicationState: "persisted",
		ProjectRoot:      root,
		TargetFile:       filepath.Join(root, "Gadget.java"),
	})
	if err != nil {
		t.Fatal(err)
	}
	result := projected.(map[string]any)
	target, ok := result["target"].(map[string]any)
	if !ok || target["file"] != "Gadget.java" {
		t.Fatalf("rename target = %#v", result["target"])
	}
}

func TestRenameContinuationConsumersMatchLiveInputFields(t *testing.T) {
	registry := operation.DefaultRegistry()
	producer, ok := registry.LookupMCP("semantic_rename")
	if !ok || producer.Output == nil {
		t.Fatal("rename output contract missing")
	}
	entries := registry.All()
	for _, continuation := range producer.Output.Continuations {
		if renameContinuationField(continuation.Field) {
			assertRenameContinuationConsumers(t, registry, entries, producer.MCPName, continuation)
		}
	}
}

func renameContinuationField(field string) bool {
	return field == "file" || field == "symbol" || field == "expected_revision"
}

func assertRenameContinuationConsumers(t *testing.T, registry *operation.Registry, entries []operation.Entry, producer string, continuation operation.OutputContinuation) {
	t.Helper()
	source := producer + "." + continuation.Field
	assertRenameConsumerEdges(t, entries, continuation, source)
	assertDeclaredRenameConsumers(t, registry, continuation, source)
}

func assertRenameConsumerEdges(t *testing.T, entries []operation.Entry, continuation operation.OutputContinuation, source string) {
	t.Helper()
	for _, consumer := range entries {
		for _, parameter := range consumer.Params {
			acceptsSource := parameter.Role == continuation.Role && slices.Contains(parameter.SourceFields, source)
			if acceptsSource && !slices.Contains(continuation.Consumers, consumer.MCPName) {
				t.Errorf("%s.%s accepts %s without a declared continuation consumer edge", consumer.MCPName, parameter.JSONName, source)
			}
		}
	}
}

func assertDeclaredRenameConsumers(t *testing.T, registry *operation.Registry, continuation operation.OutputContinuation, source string) {
	t.Helper()
	for _, consumerName := range continuation.Consumers {
		consumer, exists := registry.LookupMCP(consumerName)
		if !exists {
			t.Errorf("%s continuation names unknown consumer %q", continuation.Field, consumerName)
			continue
		}
		if !renameConsumerAcceptsSource(consumer, continuation, source) {
			t.Errorf("%s continuation names incompatible consumer %q", continuation.Field, consumerName)
		}
	}
}

func renameConsumerAcceptsSource(consumer operation.Entry, continuation operation.OutputContinuation, source string) bool {
	for _, parameter := range consumer.Params {
		if slices.Contains(parameter.SourceFields, source) && (parameter.Role == "" || parameter.Role == continuation.Role) {
			return true
		}
	}
	return false
}

func TestVerifyPolicyKeepsNilUnrestrictedAndEmptyFailClosed(t *testing.T) {
	newRoot := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		for name, source := range map[string]string{
			"go.mod":  "module example.com/policy\n\ngo 1.24\n",
			"main.go": "package main\nfunc main(){ }\n",
		} {
			if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return root
	}
	languageBackend := &verifyPolicyGoBackend{}
	backendRegistry, err := backend.NewRegistry(languageBackend)
	if err != nil {
		t.Fatal(err)
	}
	service := backend.NewService(backendRegistry)
	registry := operation.DefaultRegistry()
	entry, ok := registry.LookupCLI("verify")
	if !ok {
		t.Fatal("verify operation is not registered")
	}
	request := map[string]any{"path": ".", "language": "auto"}
	for _, test := range []struct {
		name          string
		root          string
		allowed       []backend.LanguageID
		wantFailure   bool
		wantFormatted bool
	}{
		{name: "nil is unrestricted", root: newRoot(t), wantFormatted: true},
		{name: "empty denies all", root: newRoot(t), allowed: []backend.LanguageID{}, wantFailure: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			call := operation.CallContext{
				Ctx: context.Background(), WorkDir: test.root,
				Project: backend.ProjectContext{RootDir: test.root}, Registry: registry, Service: service,
				AllowedLanguages: test.allowed,
			}
			result, err := registry.Dispatch(call, entry.Key, request)
			if err != nil {
				t.Fatal(err)
			}
			verify := result.(operation.VerifyRes)
			if (verify.Failure != "") != test.wantFailure {
				t.Fatalf("verify result = %+v", verify)
			}
			contents, err := os.ReadFile(filepath.Join(test.root, "main.go"))
			if err != nil {
				t.Fatal(err)
			}
			formatted := strings.Contains(string(contents), "func main() {}")
			if formatted != test.wantFormatted {
				t.Fatalf("source formatted=%t, want %t: %s", formatted, test.wantFormatted, contents)
			}
			if test.wantFailure && (len(verify.Coverage) != 1 || verify.Coverage[0].Language != "go" || verify.Coverage[0].Status != "disabled") {
				t.Fatalf("empty policy coverage = %+v, want Go disabled", verify.Coverage)
			}
		})
	}
	if languageBackend.calls != 1 {
		t.Fatalf("Go backend calls = %d, want one call for nil policy and none for empty policy", languageBackend.calls)
	}
}

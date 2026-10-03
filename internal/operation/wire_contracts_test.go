// WHY: This file exercises contract handlers against temporary Go modules to prove overlay preflight and no-publication refusals.

package operation

import (
	"bytes"
	"context"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"semedit/internal/contractplan"
	"semedit/internal/sourcebuf"
	"slices"
	"strings"
	"testing"
)

func TestImplementContractRefusesShadowedBuiltinPanicWithoutPublishing(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		extraFiles map[string]string
	}{
		{
			name:   "package declaration",
			source: "package demo\n\ntype Worker struct{}\nfunc panic(any) {}\ntype Contract interface { Read() }\n",
		},
		{
			name:       "file import",
			source:     "package demo\n\nimport panic \"example.com/contracts/other\"\nvar _ = panic.Value\ntype Worker struct{}\ntype Contract interface { Read() }\n",
			extraFiles: map[string]string{"other/other.go": "package other\n\nconst Value = 1\n"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/contracts\n\ngo 1.27.1\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			for name, source := range test.extraFiles {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			file := filepath.Join(root, "types.go")
			before := []byte(test.source)
			if err := os.WriteFile(file, before, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := runImplementContract(context.Background(), CallContext{WorkDir: root}, implementContractRequest{
				File: "types.go", Symbol: "Worker", Contract: "Contract",
				ReceiverKind: contractplan.ReceiverPointer, BodyPolicy: contractplan.BodyPolicyPanicStub,
			})
			if err == nil || !strings.Contains(err.Error(), "panic_stub would resolve panic") {
				t.Fatalf("expected builtin shadow refusal, got %v", err)
			}
			after, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(after, before) {
				t.Fatalf("source changed after refusal: %q", after)
			}
		})
	}
}

func TestFinalizedSignatureTargetRequiresTheSurvivingFunction(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantLive bool
	}{
		{name: "function", source: "package demo\nfunc write() {}\n", wantLive: true},
		{name: "same-name type only", source: "package demo\ntype write struct{}\n"},
		{name: "method only", source: "package demo\ntype Writer struct{}\nfunc (Writer) write() {}\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "write.go")
			if err := os.WriteFile(path, []byte(test.source), 0o600); err != nil {
				t.Fatal(err)
			}
			revision := structuralRevision([]byte(test.source))
			candidate := SymbolTarget{File: "write.go", Symbol: "write", ExpectedRevision: &revision}
			typeCandidate := SymbolTarget{File: "write.go", Symbol: "writeOptions", ExpectedRevision: &revision}
			result := structuralPlanResult{
				Edit:    FileEditRes{Display: "write.go", Symbol: "write"},
				Receipt: EditReceipt{Status: EditPersisted, Continuations: []SymbolTarget{typeCandidate, candidate}},
			}
			target, err := finalizedSignatureFunctionTarget(context.Background(), CallContext{WorkDir: root}, result)
			if err != nil {
				t.Fatalf("finalizedSignatureFunctionTarget() error = %v", err)
			}
			if (target != nil) != test.wantLive {
				t.Fatalf("finalized target = %#v, want live %v", target, test.wantLive)
			}
		})
	}
	staleRevision := "stale"
	result := structuralPlanResult{
		Edit:    FileEditRes{Display: "write.go", Symbol: "write"},
		Receipt: EditReceipt{Status: EditPersisted, Continuations: []SymbolTarget{{File: "write.go", Symbol: "write", ExpectedRevision: &staleRevision}}},
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "write.go"), []byte("package demo\nfunc write() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if target, err := finalizedSignatureFunctionTarget(context.Background(), CallContext{WorkDir: root}, result); err != nil || target != nil {
		t.Fatalf("stale finalized target = %#v, %v; want no target", target, err)
	}
	result.Receipt.Status = EditAppliedWorking
	result.Receipt.Continuations = nil
	if target, err := finalizedSignatureFunctionTarget(context.Background(), CallContext{WorkDir: root, InBatch: true}, result); err != nil || target != nil {
		t.Fatalf("buffered finalized target = %#v, %v; want no live target", target, err)
	}
}

func TestSignatureProjectionUsesOnlyTypedFunctionTarget(t *testing.T) {
	typeCandidate := SymbolTarget{File: "write.go", Symbol: "writeOptions"}
	functionRevision := "rev-2"
	functionCandidate := SymbolTarget{File: "write.go", Symbol: "write", ExpectedRevision: &functionRevision}
	result := structuralPlanResult{
		Edit:           FileEditRes{Display: "write.go", Symbol: "write"},
		Receipt:        EditReceipt{Continuations: []SymbolTarget{typeCandidate, functionCandidate}},
		FunctionTarget: &functionCandidate,
	}
	projected := projectSignatureExpansion(result).(map[string]any)
	if projected["file"] != "write.go" || projected["symbol"] != "write" || projected["function_target"] != functionCandidate {
		t.Fatalf("projected function target = %#v", projected)
	}

	result.FunctionTarget = nil
	projected = projectSignatureExpansion(result).(map[string]any)
	for _, field := range []string{"file", "symbol", "function_target"} {
		if _, exists := projected[field]; exists {
			t.Errorf("unproven primary field %q was projected: %#v", field, projected[field])
		}
	}
}

func TestSignatureContinuationContractSeparatesReceiptInspectionFromFunctionEdits(t *testing.T) {
	registry := DefaultRegistry()
	producers := []string{signatureChangeConsumer, introduceParameterConsumer, introduceParameterObjectConsumer}
	functionConsumers := append(append([]string(nil), producers...), signatureReplaceBodyConsumer)
	for _, producerName := range producers {
		producer, ok := registry.LookupMCP(producerName)
		if !ok || producer.Output == nil || producer.Output.ResultSchema == nil {
			t.Fatalf("%s output contract missing", producerName)
		}
		functionTargetSchema, ok := producer.Output.ResultSchema.Properties["function_target"]
		if !ok || slices.Contains(producer.Output.ResultSchema.Required, "function_target") || !slices.Contains(functionTargetSchema.Required, "expected_revision") {
			t.Fatalf("%s function_target schema is not optional and fully guarded: %#v", producerName, functionTargetSchema)
		}
		paths := make(map[string]OutputContinuation)
		for _, continuation := range producer.Output.Continuations {
			paths[continuation.Path] = continuation
		}
		for _, field := range []string{"file", "symbol"} {
			path := "result.edit_receipt.continuations[]." + field
			continuation, exists := paths[path]
			if !exists || !slices.Equal(continuation.Consumers, []string{"semantic_inspect_symbol"}) {
				t.Errorf("%s generic receipt path %q consumers = %#v, exists %v", producerName, path, continuation.Consumers, exists)
			}
		}
		if _, exists := paths["result.edit_receipt.continuations[].expected_revision"]; exists {
			t.Errorf("%s generic receipt revision was advertised as a consumer input", producerName)
		}
		for _, field := range []string{"file", "symbol", "expected_revision"} {
			path := "result.function_target." + field
			continuation, exists := paths[path]
			if !exists || !continuation.Optional || !slices.Equal(continuation.Consumers, functionConsumers) {
				t.Errorf("%s function target path %q consumers = %#v optional=%v exists=%v", producerName, path, continuation.Consumers, continuation.Optional, exists)
			}
		}
		for _, continuation := range producer.Output.Continuations {
			source := producerName + "." + continuation.Path
			for _, consumerName := range continuation.Consumers {
				consumer, exists := registry.LookupMCP(consumerName)
				if !exists {
					t.Errorf("%s continuation names unknown consumer %q", continuation.Path, consumerName)
					continue
				}
				compatible := false
				for _, parameter := range consumer.Params {
					if (parameter.Role == "" || parameter.Role == continuation.Role) && slices.Contains(parameter.SourceFields, source) {
						compatible = true
						break
					}
				}
				if !compatible {
					t.Errorf("%s continuation names consumer %q without a matching input field", continuation.Path, consumerName)
				}
			}
		}
	}

	inspect, ok := registry.LookupMCP("semantic_inspect_symbol")
	if !ok {
		t.Fatal("inspect tool missing")
	}
	for _, field := range []string{"file", "symbol"} {
		parameterName := field
		var parameter *ParameterContract
		for index := range inspect.Params {
			if inspect.Params[index].JSONName == parameterName {
				parameter = &inspect.Params[index]
				break
			}
		}
		if parameter == nil {
			t.Fatalf("inspect %s parameter missing", field)
		}
		for _, producerName := range producers {
			source := producerName + ".result.edit_receipt.continuations[]." + field
			if !slices.Contains(parameter.SourceFields, source) {
				t.Errorf("inspect %s omits generic receipt source %q", field, source)
			}
		}
	}
	for index := range inspect.Params {
		for _, source := range signatureReceiptSourceFields("expected_revision") {
			if slices.Contains(inspect.Params[index].SourceFields, source) {
				t.Errorf("inspect accepts unsupported signature receipt revision source %q", source)
			}
		}
	}

	replaceBody, ok := registry.LookupMCP(signatureReplaceBodyConsumer)
	if !ok {
		t.Fatal("replace-body tool missing")
	}
	for index := range replaceBody.Params {
		parameter := &replaceBody.Params[index]
		if parameter.JSONName == wireExpectedRevision {
			for _, source := range signatureTargetSourceFields(wireExpectedRevision) {
				if !slices.Contains(parameter.SourceFields, source) {
					t.Errorf("replace-body revision parameter omits function target source %q", source)
				}
			}
			for _, source := range signatureReceiptSourceFields(wireExpectedRevision) {
				if slices.Contains(parameter.SourceFields, source) {
					t.Errorf("replace-body accepts generic receipt revision source %q", source)
				}
			}
		}
	}
}

func TestImplementContractUsesCurrentBufferedSourceAndDoesNotPublish(t *testing.T) {
	root := t.TempDir()
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(canonicalRoot, "go.mod"), []byte("module example.com/contracts\n\ngo 1.27.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(canonicalRoot, "types.go")
	diskBefore := []byte("package demo\ntype Worker struct{}\n")
	if err := os.WriteFile(path, diskBefore, 0o600); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(canonicalRoot)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	if err := buffer.AuthorizeWrites(ctx, canonicalRoot); err != nil {
		t.Fatal(err)
	}
	afterChild := []byte("package demo\ntype Worker struct{}\ntype Contract interface { Read() error }\n")
	if err := buffer.Apply(ctx, []sourcebuf.Change{{Path: path, Before: diskBefore, After: afterChild}}); err != nil {
		t.Fatal(err)
	}
	result, err := runImplementContract(ctx, CallContext{Ctx: ctx, WorkDir: canonicalRoot}, implementContractRequest{
		File: "types.go", Symbol: "Worker", Contract: "Contract",
		ReceiverKind: contractplan.ReceiverPointer, BodyPolicy: contractplan.BodyPolicyPanicStub,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "working_buffer" || result.NoOp {
		t.Fatalf("buffered outcome = %#v", result)
	}
	current, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(current, []byte("type Contract interface")) || !bytes.Contains(current, []byte("func (*Worker) Read() error")) {
		t.Fatalf("current overlay does not contain input and generated declarations: %s", current)
	}
	diskAfter, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(diskAfter, diskBefore) {
		t.Fatalf("buffered operation published source to disk: %s", diskAfter)
	}
}

func TestExtractInterfaceUsesBufferedMethodAndDoesNotPublish(t *testing.T) {
	root := t.TempDir()
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(canonicalRoot, "go.mod"), []byte("module example.com/contracts\n\ngo 1.27.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(canonicalRoot, "store.go")
	diskBefore := []byte("package demo\ntype Store struct{}\n")
	if err := os.WriteFile(path, diskBefore, 0o600); err != nil {
		t.Fatal(err)
	}
	destinationPath := filepath.Join(canonicalRoot, "contracts.go")
	destinationBefore := []byte("package demo\n")
	if err := os.WriteFile(destinationPath, destinationBefore, 0o600); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(canonicalRoot)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	if err := buffer.AuthorizeWrites(ctx, canonicalRoot); err != nil {
		t.Fatal(err)
	}
	afterChild := []byte("package demo\ntype Store struct{}\nfunc (*Store) Read() error { return nil }\n")
	if err := buffer.Apply(ctx, []sourcebuf.Change{{Path: path, Before: diskBefore, After: afterChild}}); err != nil {
		t.Fatal(err)
	}
	result, err := runExtractInterface(ctx, CallContext{Ctx: ctx, WorkDir: canonicalRoot}, extractInterfaceRequest{
		File: "store.go", InType: "Store", Symbol: "StoreReader", DestinationFile: "contracts.go", ReceiverKind: contractplan.ReceiverPointer,
		Members: []contractMemberTarget{{File: "store.go", Symbol: "(*Store).Read"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "working_buffer" || result.NoOp || result.File != "contracts.go" {
		t.Fatalf("buffered outcome = %#v", result)
	}
	currentSource, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(currentSource, []byte("func (*Store) Read() error")) || bytes.Contains(currentSource, []byte("type StoreReader interface")) {
		t.Fatalf("source overlay changed unexpectedly: %s", currentSource)
	}
	currentDestination, err := sourcebuf.ReadFile(ctx, destinationPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(currentDestination, []byte("type StoreReader interface")) || !bytes.Contains(currentDestination, []byte("Read() error")) {
		t.Fatalf("destination overlay lacks extracted interface: %s", currentDestination)
	}
	diskSource, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	diskDestination, err := os.ReadFile(destinationPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(diskSource, diskBefore) || !bytes.Equal(diskDestination, destinationBefore) {
		t.Fatalf("buffered operation published source or destination to disk")
	}
}

func TestImplementContractExactSatisfactionIsNoOpWithoutPublication(t *testing.T) {
	root := t.TempDir()
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(canonicalRoot, "go.mod"), []byte("module example.com/contracts\n\ngo 1.27.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := []byte("package demo\ntype Worker struct{}\nfunc (*Worker) Read() error { return nil }\ntype Contract interface { Read() error }\n")
	path := filepath.Join(canonicalRoot, "types.go")
	if err := os.WriteFile(path, before, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := runImplementContract(context.Background(), CallContext{WorkDir: canonicalRoot}, implementContractRequest{
		File: "types.go", Symbol: "Worker", Contract: "Contract",
		ReceiverKind: contractplan.ReceiverPointer, BodyPolicy: contractplan.BodyPolicyPanicStub,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.NoOp || result.Status != "no_op" || len(result.Changes) != 0 {
		t.Fatalf("exact satisfaction outcome = %#v, want explicit no-op with no changes", result)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("no-op changed source bytes: %s", after)
	}
}

func TestContractSnapshotReadSetIncludesModuleAndExistingPackageFiles(t *testing.T) {
	root := t.TempDir()
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"go.mod":       []byte("module example.com/contracts\n\ngo 1.27.1\n"),
		"store.go":     []byte("package demo\ntype Store struct{}\n"),
		"contracts.go": []byte("package demo\ntype Contract interface { Read() error }\n"),
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(canonicalRoot, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := loadContractSnapshot(context.Background(), canonicalRoot, []string{"store.go", "contracts.go"})
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]string, len(snapshot.ReadSet))
	for _, entry := range snapshot.ReadSet {
		seen[entry.Path] = entry.Revision
	}
	for name, data := range files {
		if seen[name] != contractDigest(data) {
			t.Errorf("read set revision for %q = %q, want digest %q", name, seen[name], contractDigest(data))
		}
	}
}

func TestImplementContractRejectsStaleSelectedRevisionWithoutPublication(t *testing.T) {
	root := t.TempDir()
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(canonicalRoot, "go.mod"), []byte("module example.com/contracts\n\ngo 1.27.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := []byte("package demo\ntype Worker struct{}\ntype Contract interface { Read() error }\n")
	path := filepath.Join(canonicalRoot, "types.go")
	if err := os.WriteFile(path, before, 0o600); err != nil {
		t.Fatal(err)
	}
	stale := "old-selected-file-digest"
	_, err = runImplementContract(context.Background(), CallContext{WorkDir: canonicalRoot}, implementContractRequest{
		File: "types.go", Symbol: "Worker", Contract: "Contract", ExpectedRevision: &stale,
		ReceiverKind: contractplan.ReceiverPointer, BodyPolicy: contractplan.BodyPolicyPanicStub,
	})
	if err == nil || !strings.Contains(err.Error(), "selected source revision changed") {
		t.Fatalf("stale revision error = %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("stale revision changed source bytes: %s", after)
	}
}

func TestContractParameterParsersPreserveClosedValuesAndRevisionOmission(t *testing.T) {
	validMembers, err := parseContractMembers([]any{map[string]any{"file": "store.go", "symbol": "(*Store).Read"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(validMembers) != 1 || validMembers[0] != (contractMemberTarget{File: "store.go", Symbol: "(*Store).Read"}) {
		t.Fatalf("parsed members = %#v", validMembers)
	}
	for name, value := range map[string]any{
		"missing array": nil,
		"empty array":   []any{},
		"unknown field": []any{map[string]any{"file": "store.go", "symbol": "(*Store).Read", "extra": true}},
		"wrong object":  []any{map[string]string{"file": "store.go", "symbol": "(*Store).Read"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseContractMembers(value); err == nil {
				t.Fatalf("parseContractMembers(%#v) accepted invalid input", value)
			}
		})
	}
	omitted, err := parseExpectedContractRevision(map[string]any{})
	if err != nil || omitted == nil || *omitted != "" {
		t.Fatalf("omitted expected_revision = %v, %v", omitted, err)
	}
	for name, raw := range map[string]map[string]any{
		"null":  {"expected_revision": nil},
		"empty": {"expected_revision": ""},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseExpectedContractRevision(raw); err == nil {
				t.Fatalf("parseExpectedContractRevision(%#v) accepted invalid input", raw)
			}
		})
	}
}

func TestContractParserRetainsAnalyzedBytesAcrossDiskDrift(t *testing.T) {
	root := t.TempDir()
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	root = canonicalRoot
	path := filepath.Join(root, "source.go")
	before := []byte("package sample\ntype Before struct{}\n")
	after := []byte("package sample\ntype After struct{}\n")
	if err := os.WriteFile(path, before, 0o600); err != nil {
		t.Fatal(err)
	}
	captured := make(map[string][]byte)
	file, err := parseContractSource(context.Background(), root, token.NewFileSet(), path, before, captured, func(canonical string) {
		if err := os.WriteFile(canonical, after, 0o600); err != nil {
			t.Fatalf("change source after parser capture: %v", err)
		}
	})
	if err != nil {
		t.Fatalf("parse captured source: %v", err)
	}
	if len(file.Decls) != 1 {
		t.Fatalf("parsed declarations = %d, want 1", len(file.Decls))
	}
	declaration, ok := file.Decls[0].(*ast.GenDecl)
	if !ok || declaration.Specs[0].(*ast.TypeSpec).Name.Name != "Before" {
		t.Fatalf("parser did not use captured pre-drift source: %#v", file.Decls)
	}
	if !bytes.Equal(captured[path], before) {
		t.Fatalf("captured input = %q, want parsed source %q", captured[path], before)
	}
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(current, after) {
		t.Fatalf("drift fixture source = %q, want %q", current, after)
	}
}

func TestImplementContractPreflightsSamePackageTestVariants(t *testing.T) {
	for _, test := range []struct {
		name           string
		testSource     string
		symlinkSource  string
		symlinkInRoot  bool
		wantErr        bool
		wantErrMessage string
	}{
		{name: "compiles with tests", testSource: "package demo\n\nimport \"testing\"\n\nfunc TestWorker(t *testing.T) {}\n", wantErr: false},
		{name: "refuses test method collision", testSource: "package demo\n\nfunc (*Worker) Run() error { return nil }\n", wantErr: true, wantErrMessage: "test variant"},
		{name: "refuses in-root symlinked test method collision", symlinkSource: "package demo\n\nfunc (*Worker) Run() error { return nil }\n", symlinkInRoot: true, wantErr: true, wantErrMessage: "does not support test symlinks"},
		{name: "refuses out-of-root symlinked test method collision", symlinkSource: "package demo\n\nfunc (*Worker) Run() error { return nil }\n", wantErr: true, wantErrMessage: "outside the snapshot root"},
		{name: "refuses out-of-root symlinked test source without collision", symlinkSource: "package demo\n\nimport \"testing\"\n\nfunc TestWorker(t *testing.T) {}\n", wantErr: true, wantErrMessage: "outside the snapshot root"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for name, source := range map[string]string{
				"go.mod":     "module example.com/testvariant\n\ngo 1.27.1\n",
				"types.go":   "package demo\n\ntype Worker struct{}\n\ntype Runner interface { Run() error }\n",
				"methods.go": "package demo\n",
			} {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if test.symlinkSource != "" {
				targetRoot := t.TempDir()
				if test.symlinkInRoot {
					targetRoot = filepath.Join(root, "fixtures")
					if err := os.MkdirAll(targetRoot, 0o700); err != nil {
						t.Fatal(err)
					}
				}
				externalTest := filepath.Join(targetRoot, "external_test.go")
				if err := os.WriteFile(externalTest, []byte(test.symlinkSource), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(externalTest, filepath.Join(root, "worker_test.go")); err != nil {
					t.Fatalf("create test source symlink: %v", err)
				}
			} else if test.testSource != "" {
				if err := os.WriteFile(filepath.Join(root, "worker_test.go"), []byte(test.testSource), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.ReadFile(filepath.Join(root, "methods.go"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := runImplementContract(context.Background(), CallContext{WorkDir: root}, implementContractRequest{
				File: "types.go", Symbol: "Worker", Contract: "Runner", DestinationFile: "methods.go",
				ReceiverKind: contractplan.ReceiverPointer, BodyPolicy: contractplan.BodyPolicyPanicStub,
			})
			switch {
			case test.wantErr:
				if err == nil || !strings.Contains(err.Error(), test.wantErrMessage) {
					t.Fatalf("expected refusal containing %q, got %v", test.wantErrMessage, err)
				}
				after, readErr := os.ReadFile(filepath.Join(root, "methods.go"))
				if readErr != nil {
					t.Fatal(readErr)
				}
				if !bytes.Equal(after, before) {
					t.Fatalf("destination changed after test variant refusal: %q", after)
				}
			case err != nil:
				t.Fatalf("valid test variant rejected: %v", err)
			default:
				wantRevision := contractDigest([]byte(test.testSource))
				foundTestRevision := false
				for _, revision := range result.ReadSet {
					if revision.Path == "worker_test.go" && revision.Revision == wantRevision {
						foundTestRevision = true
						break
					}
				}
				if !foundTestRevision {
					t.Fatalf("successful contract preflight read set lacks worker_test.go revision %q: %#v", wantRevision, result.ReadSet)
				}
			}
		})
	}
}

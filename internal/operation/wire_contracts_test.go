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

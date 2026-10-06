// Registry-derived CLI scenarios prove that declared broken-code combinations match observable Go operations.

package main

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"semedit/internal/atomicfile"
	"semedit/internal/backend"
	"semedit/internal/capability"
	"semedit/internal/operation"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"golang.org/x/tools/txtar"
)

func TestREQ001BrokenCodeBitmapCLIContracts(t *testing.T) {
	registry := operation.DefaultRegistry()
	scriptDir := t.TempDir()
	for faults := capability.CodeFaults(0); faults <= capability.AllCodeFaults; faults++ {
		archive := brokenCodeContractArchive(t, registry, faults)
		path := filepath.Join(scriptDir, fmt.Sprintf("faults_%02x.txtar", uint8(faults)))
		if err := atomicfile.WriteAtomic(path, txtar.Format(archive)); err != nil {
			t.Fatalf("write generated CLI case: %v", err)
		}
	}
	testscript.Run(t, testscript.Params{
		Dir: scriptDir,
		Setup: func(env *testscript.Env) error {
			for _, key := range []string{"GOPATH", "GOCACHE", "GOROOT", "HOME", "GOBIN", "GOENV", "GOMODCACHE", "GOTMPDIR", "GOWORK", "GOPLSCACHE"} {
				if value := os.Getenv(key); value != "" {
					env.Vars = append(env.Vars, key+"="+value)
				}
			}
			env.Vars = append(env.Vars, "PATH="+os.Getenv("PATH"))
			return nil
		},
	})
}

func brokenCodeFixtureSource(t *testing.T, faults, syntaxFault, typeFault capability.CodeFaults, packageName string, selected bool, returnValue int) string {
	t.Helper()
	var source strings.Builder
	fmt.Fprintf(&source, "package %s\n\n", packageName)
	if selected {
		fmt.Fprintf(&source, "func Chosen() int {\n\treturn %d\n}\n", returnValue)
	}
	if faults&typeFault != 0 {
		fmt.Fprintf(&source, "\nvar failure%d = missing%d\n", uint8(typeFault), uint8(typeFault))
	}
	if faults&syntaxFault != 0 {
		fmt.Fprintf(&source, "\nfunc malformed%d(\n", uint8(syntaxFault))
		return source.String()
	}
	formatted, err := format.Source([]byte(source.String()))
	if err != nil {
		t.Fatalf("format generated valid-syntax fixture: %v", err)
	}
	return string(formatted)
}

func brokenCodeContractArchive(t *testing.T, registry *operation.Registry, faults capability.CodeFaults) *txtar.Archive {
	t.Helper()
	archive := &txtar.Archive{}
	var commands strings.Builder
	fmt.Fprintf(&commands, "# Registry-derived exact fault combination: %s.\n", faults)
	target := brokenCodeFixtureSource(t, faults, capability.FaultTargetSyntax, capability.FaultTargetType, "target", true, 1)
	archive.Files = []txtar.File{
		{Name: "go.mod", Data: []byte("module example.com/broken-contract\n\ngo 1.23\n")},
		{Name: "target.go", Data: []byte(target)},
		{Name: "before/target.txt", Data: []byte(target)},
		{Name: "sibling.go", Data: []byte(brokenCodeFixtureSource(t, faults, capability.FaultSamePackageSyntax, capability.FaultSamePackageType, "target", false, 1))},
		{Name: "independent/other.go", Data: []byte(brokenCodeFixtureSource(t, faults, capability.FaultOtherPackageSyntax, capability.FaultOtherPackageType, "independent", false, 1))},
		{Name: "body.txt", Data: []byte("return 2\n")},
	}
	archive.Files = append(archive.Files,
		txtar.File{Name: "before/sibling.txt", Data: archive.Files[3].Data},
		txtar.File{Name: "before/other.txt", Data: archive.Files[4].Data})
	for _, key := range []string{capability.OpLookup, capability.OpOutline, capability.OpInspect, "replace_body"} {
		appendBrokenCodeContractOperation(t, registry, archive, &commands, key, faults)
	}
	archive.Comment = []byte(commands.String())
	return archive
}

func appendBrokenCodeContractOperation(t *testing.T, registry *operation.Registry, archive *txtar.Archive, commands *strings.Builder, key string, faults capability.CodeFaults) {
	t.Helper()
	entry, ok := registry.LookupKey(key)
	if !ok {
		t.Fatalf("missing registry entry %q", key)
	}
	contract, ok := entry.LanguageContracts[backend.LanguageGo]
	if !ok {
		t.Fatalf("missing Go contract for %q", key)
	}
	variant, ok := contract.Variant(capability.VariantSelectedFile)
	if !ok {
		t.Fatalf("missing selected-file variant for %q", key)
	}
	cases, err := variant.ErrorTolerance.Cases()
	if err != nil {
		t.Fatalf("%s cases: %v", key, err)
	}
	support := cases[int(faults)].Support
	if support == capability.FaultUnknown || support == capability.FaultConditional {
		return
	}
	command := "exec semedit " + entry.CLIName
	if key == capability.OpOutline {
		command += " --path target.go"
	} else {
		command += " --file target.go --symbol Chosen"
	}
	if key == "replace_body" {
		commands.WriteString("stdin body.txt\n")
		command += " --body - --auto-organize-imports=false"
	}
	if support == capability.FaultRejected {
		fmt.Fprintf(commands, "! %s\nstderr '.'\ncmp target.go before/target.txt\n", command)
	} else {
		fmt.Fprintf(commands, "%s\nstdout 'Chosen'\n", command)
		if key == "replace_body" {
			archive.Files = append(archive.Files, txtar.File{Name: "want/target.txt", Data: []byte(brokenCodeFixtureSource(t, faults, capability.FaultTargetSyntax, capability.FaultTargetType, "target", true, 2))})
			commands.WriteString("cmp target.go want/target.txt\n")
		} else {
			commands.WriteString("cmp target.go before/target.txt\n")
		}
	}
	commands.WriteString("cmp sibling.go before/sibling.txt\ncmp independent/other.go before/other.txt\n")
}

// Package astedit_test validates AST declaration insertion logic and placement qualifiers.
package astedit_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/astedit"
)

const baseFile = `package testpkg

import (
	"fmt"
)

func ExportedOne() {
	fmt.Println("one")
}

func ExportedTwo() {
	fmt.Println("two")
}

func unexportedOne() {
	fmt.Println("priv one")
}

func unexportedTwo() {
	fmt.Println("priv two")
}
`

func setupTestFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "target.go")
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		t.Fatalf("write setup file: %v", err)
	}
	return target
}

func TestInsertDeclaration_FileStartAndEnd(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("file_start", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, baseFile)
		snippet := "const EarlyConst = 100"
		err := astedit.InsertDeclaration(ctx, file, snippet, astedit.Options{
			Placement: astedit.PlacementFileStart,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := os.ReadFile(filepath.Clean(file))
		content := string(data)
		idxConst := strings.Index(content, "const EarlyConst = 100")
		idxImport := strings.Index(content, `"fmt"`)
		idxExported := strings.Index(content, "func ExportedOne()")
		if idxConst < idxImport || idxConst > idxExported {
			t.Errorf("expected EarlyConst between import and ExportedOne, got:\n%s", content)
		}
	})

	t.Run("file_end", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, baseFile)
		snippet := "func LastFunc() {}"
		err := astedit.InsertDeclaration(ctx, file, snippet, astedit.Options{
			Placement: astedit.PlacementFileEnd,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := os.ReadFile(filepath.Clean(file))
		content := string(data)
		idxLast := strings.Index(content, "func LastFunc()")
		idxPrivTwo := strings.Index(content, "func unexportedTwo()")
		if idxLast < idxPrivTwo {
			t.Errorf("expected LastFunc after unexportedTwo, got:\n%s", content)
		}
	})
}

func TestInsertDeclaration_PublicAndPrivateSections(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("public_start", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, baseFile)
		snippet := "func FirstPublic() {}"
		err := astedit.InsertDeclaration(ctx, file, snippet, astedit.Options{
			Placement: astedit.PlacementPublicStart,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := os.ReadFile(filepath.Clean(file))
		content := string(data)
		idxNew := strings.Index(content, "func FirstPublic()")
		idxOne := strings.Index(content, "func ExportedOne()")
		if idxNew > idxOne {
			t.Errorf("expected FirstPublic before ExportedOne, got:\n%s", content)
		}
	})

	t.Run("public_end", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, baseFile)
		snippet := "func LastPublic() {}"
		err := astedit.InsertDeclaration(ctx, file, snippet, astedit.Options{
			Placement: astedit.PlacementPublicEnd,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := os.ReadFile(filepath.Clean(file))
		content := string(data)
		idxNew := strings.Index(content, "func LastPublic()")
		idxTwo := strings.Index(content, "func ExportedTwo()")
		idxPrivOne := strings.Index(content, "func unexportedOne()")
		if idxNew < idxTwo || idxNew > idxPrivOne {
			t.Errorf("expected LastPublic between ExportedTwo and unexportedOne, got:\n%s", content)
		}
	})

	t.Run("private_start", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, baseFile)
		snippet := "func firstPrivate() {}"
		err := astedit.InsertDeclaration(ctx, file, snippet, astedit.Options{
			Placement: astedit.PlacementPrivateStart,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := os.ReadFile(filepath.Clean(file))
		content := string(data)
		idxNew := strings.Index(content, "func firstPrivate()")
		idxPrivOne := strings.Index(content, "func unexportedOne()")
		if idxNew > idxPrivOne {
			t.Errorf("expected firstPrivate before unexportedOne, got:\n%s", content)
		}
	})

	t.Run("public_section_fallback_when_only_private_exists", func(t *testing.T) {
		t.Parallel()
		onlyPrivate := `package testpkg

func privOnly() {}
`
		file := setupTestFile(t, onlyPrivate)
		snippet := "func ExportedNew() {}"
		err := astedit.InsertDeclaration(ctx, file, snippet, astedit.Options{
			Placement: astedit.PlacementPublicStart,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := os.ReadFile(filepath.Clean(file))
		content := string(data)
		idxNew := strings.Index(content, "func ExportedNew()")
		idxPriv := strings.Index(content, "func privOnly()")
		if idxNew > idxPriv {
			t.Errorf("expected ExportedNew before privOnly due to public-precedes-private invariant, got:\n%s", content)
		}
	})
}

func TestInsertDeclaration_SymbolRelative(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("before_symbol", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, baseFile)
		snippet := "func PreExportedTwo() {}"
		err := astedit.InsertDeclaration(ctx, file, snippet, astedit.Options{
			Placement:    astedit.PlacementBeforeSymbol,
			TargetSymbol: "ExportedTwo",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := os.ReadFile(filepath.Clean(file))
		content := string(data)
		idxNew := strings.Index(content, "func PreExportedTwo()")
		idxTarget := strings.Index(content, "func ExportedTwo()")
		if idxNew > idxTarget {
			t.Errorf("expected PreExportedTwo before ExportedTwo, got:\n%s", content)
		}
	})

	t.Run("after_symbol", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, baseFile)
		snippet := "func PostExportedOne() {}"
		err := astedit.InsertDeclaration(ctx, file, snippet, astedit.Options{
			Placement:    astedit.PlacementAfterSymbol,
			TargetSymbol: "ExportedOne",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := os.ReadFile(filepath.Clean(file))
		content := string(data)
		idxNew := strings.Index(content, "func PostExportedOne()")
		idxOne := strings.Index(content, "func ExportedOne()")
		idxTwo := strings.Index(content, "func ExportedTwo()")
		if idxNew < idxOne || idxNew > idxTwo {
			t.Errorf("expected PostExportedOne between ExportedOne and ExportedTwo, got:\n%s", content)
		}
	})
}

func TestInsertDeclaration_Validation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("visibility_mismatch_public_requested", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, baseFile)
		snippet := "func privateHelper() {}"
		err := astedit.InsertDeclaration(ctx, file, snippet, astedit.Options{
			Placement:  astedit.PlacementFileEnd,
			Visibility: "public",
		})
		if err == nil {
			t.Fatal("expected error on public visibility mismatch, got nil")
		}
	})

	t.Run("visibility_mismatch_private_requested", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, baseFile)
		snippet := "func PublicHelper() {}"
		err := astedit.InsertDeclaration(ctx, file, snippet, astedit.Options{
			Placement:  astedit.PlacementFileEnd,
			Visibility: "private",
		})
		if err == nil {
			t.Fatal("expected error on private visibility mismatch, got nil")
		}
	})

	t.Run("syntax_error_zero_disk_mutation", func(t *testing.T) {
		t.Parallel()
		file := setupTestFile(t, baseFile)
		snippet := "func broken( { return"
		err := astedit.InsertDeclaration(ctx, file, snippet, astedit.Options{
			Placement: astedit.PlacementFileEnd,
		})
		if err == nil {
			t.Fatal("expected syntax error, got nil")
		}
		data, _ := os.ReadFile(filepath.Clean(file))
		if string(data) != baseFile {
			t.Errorf("file was mutated on invalid syntax error:\n%s", string(data))
		}
	})
}

func TestInsertDeclaration_AutoOrganizeImports(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	file := setupTestFile(t, baseFile)
	// Snippet uses os.Getenv which is not currently imported in baseFile
	snippet := `func ReadEnv() string {
	return os.Getenv("MODE")
}`

	err := astedit.InsertDeclaration(ctx, file, snippet, astedit.Options{
		Placement:           astedit.PlacementFileEnd,
		AutoOrganizeImports: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Clean(file))
	content := string(data)
	if !strings.Contains(content, `"os"`) {
		t.Errorf("expected 'os' import to be automatically added, got:\n%s", content)
	}
	if !strings.Contains(content, "func ReadEnv()") {
		t.Errorf("expected ReadEnv to be present, got:\n%s", content)
	}
}

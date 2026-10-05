// Package astedit_test validates AST declaration insertion logic and placement qualifiers.
package astedit_test

import (
	"context"
	"errors"
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
	for _, tc := range []struct {
		name, snippet string
		placement     astedit.Placement
		before, after string
	}{
		{name: "public_start", snippet: "func FirstPublic() {}", placement: astedit.PlacementPublicStart, before: "func ExportedOne()"},
		{name: "public_end", snippet: "func LastPublic() {}", placement: astedit.PlacementPublicEnd, before: "func unexportedOne()", after: "func ExportedTwo()"},
		{name: "private_start", snippet: "func firstPrivate() {}", placement: astedit.PlacementPrivateStart, before: "func unexportedOne()"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertInsertedBetween(t, baseFile, tc.snippet, tc.placement, tc.after, tc.before)
		})
	}
	t.Run("public_section_fallback_when_only_private_exists", func(t *testing.T) {
		t.Parallel()
		assertInsertedBetween(t, "package testpkg\n\nfunc privOnly() {}\n", "func ExportedNew() {}", astedit.PlacementPublicStart, "", "func privOnly()")
	})
}

func assertInsertedBetween(t *testing.T, source, snippet string, placement astedit.Placement, after, before string) {
	t.Helper()
	file := setupTestFile(t, source)
	if err := astedit.InsertDeclaration(context.Background(), file, snippet, astedit.Options{Placement: placement}); err != nil {
		t.Fatalf("InsertDeclaration() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	name, _, hasCall := strings.Cut(snippet, "(")
	if !hasCall {
		t.Fatalf("test snippet has no declaration name: %q", snippet)
	}
	inserted := strings.Index(content, name)
	afterIndex, beforeIndex := 0, len(content)
	if after != "" {
		afterIndex = strings.Index(content, after)
	}
	if before != "" {
		beforeIndex = strings.Index(content, before)
	}
	if inserted < 0 || afterIndex < 0 || beforeIndex < 0 || inserted < afterIndex || inserted > beforeIndex {
		t.Fatalf("inserted %q outside expected bounds %q..%q:\n%s", snippet, after, before, content)
	}
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
	t.Run("visibility_mismatch_public_requested", func(t *testing.T) { assertVisibilityMismatch(t, "func privateHelper() {}", "public", "privateHelper") })
	t.Run("visibility_mismatch_private_requested", func(t *testing.T) { assertVisibilityMismatch(t, "func PublicHelper() {}", "private", "PublicHelper") })
	t.Run("missing_target_symbol", func(t *testing.T) { assertMissingTargetSymbol(t) })
	t.Run("syntax_error_zero_disk_mutation", func(t *testing.T) { assertSyntaxErrorDoesNotMutate(t) })
}

func assertVisibilityMismatch(t *testing.T, snippet, visibility, identifier string) {
	t.Helper()
	file := setupTestFile(t, baseFile)
	err := astedit.InsertDeclaration(context.Background(), file, snippet, astedit.Options{Placement: astedit.PlacementFileEnd, Visibility: visibility})
	if !errors.Is(err, astedit.ErrVisibilityMismatch) {
		t.Fatalf("expected ErrVisibilityMismatch, got %v", err)
	}
	var visibilityError *astedit.VisibilityMismatchError
	if !errors.As(err, &visibilityError) || visibilityError.Identifier != identifier {
		t.Fatalf("visibility error = %#v, want identifier %q", visibilityError, identifier)
	}
}

func assertMissingTargetSymbol(t *testing.T) {
	t.Helper()
	file := setupTestFile(t, baseFile)
	err := astedit.InsertDeclaration(context.Background(), file, "func NewFunc() {}", astedit.Options{Placement: astedit.PlacementBeforeSymbol, TargetSymbol: "NoSuchSymbol"})
	var placementError *astedit.PlacementError
	if !errors.Is(err, astedit.ErrSymbolNotFound) || !errors.As(err, &placementError) || placementError.TargetSymbol != "NoSuchSymbol" || placementError.Strategy != astedit.PlacementBeforeSymbol {
		t.Fatalf("placement error = %#v, want missing NoSuchSymbol before symbol", placementError)
	}
}

func assertSyntaxErrorDoesNotMutate(t *testing.T) {
	t.Helper()
	file := setupTestFile(t, baseFile)
	snippet := "func broken( { return"
	err := astedit.InsertDeclaration(context.Background(), file, snippet, astedit.Options{Placement: astedit.PlacementFileEnd})
	var syntaxError *astedit.SyntaxError
	if !errors.Is(err, astedit.ErrSyntax) || !errors.As(err, &syntaxError) || !syntaxError.Pos.IsValid() || syntaxError.Snippet != snippet {
		t.Fatalf("syntax error = %#v, want located source failure", syntaxError)
	}
	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != baseFile {
		t.Fatalf("file mutated on syntax error:\n%s", data)
	}
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

func TestInsertDeclarationRejectsInvalidCombinedSyntax(t *testing.T) {
	for _, organize := range []bool{false, true} {
		name := "format"
		if organize {
			name = "organize imports"
		}
		t.Run(name, func(t *testing.T) {
			original := "package api\n\nimport \"strings\"\n\nfunc Normalize(value string) string { return strings.TrimSpace(value) }\n"
			file := setupTestFile(t, original)
			before, err := os.Stat(file)
			if err != nil {
				t.Fatal(err)
			}
			err = astedit.InsertDeclaration(t.Context(), file, `import "strings"`, astedit.Options{
				Placement: astedit.PlacementFileEnd, AutoOrganizeImports: organize,
			})
			if err == nil || !strings.Contains(err.Error(), "imports must appear before other declarations") {
				t.Fatalf("wanted import ordering rejection, got %v", err)
			}
			got, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != original {
				t.Fatalf("rejected insertion changed source:\n%s", got)
			}
			after, err := os.Stat(file)
			if err != nil {
				t.Fatal(err)
			}
			if !before.ModTime().Equal(after.ModTime()) {
				t.Fatal("rejected insertion advanced the file revision")
			}
		})
	}
}

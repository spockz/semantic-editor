// WHY: Verifies explicit import requests preserve source layout and reject unsafe selector strings.
package java

import (
	"bytes"
	"testing"

	"github.com/spockz/semantic-editor/internal/javacst"
)

func TestApplyJavaExplicitImports(t *testing.T) {
	source := []byte("package example;\n\nimport java.util.List;\n\nclass Example {}\n")
	updated, err := applyJavaExplicitImports(source, []string{"java.util.Map"}, []string{"java.util.List"})
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("package example;\n\nimport java.util.Map;\n\nclass Example {}\n")
	if !bytes.Equal(updated, want) {
		t.Fatalf("imports = %q, want %q", updated, want)
	}
}

func TestApplyJavaExplicitImportsRejectsInvalidNames(t *testing.T) {
	for _, name := range []string{"", "java.util.*", "java.lang.String;", "java.util.Map as M"} {
		if _, err := applyJavaExplicitImports([]byte("class Example {}\n"), []string{name}, nil); err == nil {
			t.Errorf("invalid import %q was accepted", name)
		}
	}
}

func TestRestoreJavaImportsFromPreimage(t *testing.T) {
	preimage := []byte("package example;\n\n/*\nimport static com.acme.Kind.CREDIT;\n*/\nimport static com.acme.Kind.CREDIT; /* retained declaration */\nimport static com.acme.Kind.*;\nimport java.util.*;\n\nclass Example {}\n")
	source := []byte("package example;\n\n/*\nimport static com.acme.Kind.CREDIT;\n*/\nclass Example { Object value = CREDIT; }\n")
	updated, err := restoreJavaImportsFromPreimage(source, preimage)
	if err != nil {
		t.Fatal(err)
	}
	imports, err := javacst.ParseJavaImports(updated)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"static:com\x00.\x00acme\x00.\x00Kind\x00.\x00CREDIT",
		"static:com\x00.\x00acme\x00.\x00Kind\x00.\x00*",
		"ordinary:java\x00.\x00util\x00.\x00*",
	}
	for _, expected := range want {
		count := 0
		for _, declaration := range imports.Declarations {
			if declaration.Identity == expected {
				count++
			}
		}
		if count != 1 {
			t.Errorf("import identity %q count = %d, want 1; imports: %#v", expected, count, imports.Declarations)
		}
	}
}

func TestRemoveJavaImportPreservesSameLineDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "following type",
			source: "import java.util.List; class Example {}\n",
			want:   "class Example {}\n",
		},
		{
			name:   "following import",
			source: "import java.util.List; import java.util.Map;\nclass Example { Map value; }\n",
			want:   "import java.util.Map;\nclass Example { Map value; }\n",
		},
		{
			name:   "duplicate imports",
			source: "import java.util.List;\nimport java.util.List;\nclass Example {}\n",
			want:   "class Example {}\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := removeJavaImport([]byte(test.source), "java.util.List")
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Fatalf("source after removal = %q, want %q", got, test.want)
			}
		})
	}
}

func TestInsertJavaImportUsesExactDeclarationBoundary(t *testing.T) {
	source := []byte("package example;\nimport java.util.List; /* trailing\n * block comment\n */\nclass Example {}\n")
	updated, err := insertJavaImport(source, "java.util.Map")
	if err != nil {
		t.Fatal(err)
	}
	imports, err := javacst.ParseJavaImports(updated)
	if err != nil {
		t.Fatal(err)
	}
	if len(imports.Declarations) != 2 {
		t.Fatalf("declarations = %#v, want both imports", imports.Declarations)
	}
	if !bytes.Contains(updated, []byte("/* trailing\n * block comment\n */\nclass Example {}")) {
		t.Fatalf("trailing comment or class was changed: %q", updated)
	}
}

func TestJavaImportIdentitySeparatesStaticAndOrdinaryImports(t *testing.T) {
	static := javacst.JavaImportIdentity("p.Type.VALUE", true)
	ordinary := javacst.JavaImportIdentity("staticp.Type.VALUE", false)
	if static == ordinary {
		t.Fatalf("static and ordinary import identities collide: %q", static)
	}
}

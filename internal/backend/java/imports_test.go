// WHY: Verifies explicit import requests preserve source layout and reject unsafe selector strings.
package java

import (
	"bytes"
	"testing"
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

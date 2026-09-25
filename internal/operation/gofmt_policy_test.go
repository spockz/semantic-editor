// Package operation tests the source-selection boundary for pre-verification gofmt listing.
package operation

import (
	"testing"

	"semedit/internal/backend"
)

func TestShouldRunGofmtPreservesAutoDirectoryAndSkipsExternalFiles(t *testing.T) {
	for _, test := range []struct {
		name    string
		project backend.ProjectContext
		path    string
		want    bool
	}{
		{name: "auto directory", project: backend.ProjectContext{Language: backend.LanguageAuto}, path: ".", want: true},
		{name: "auto Go file", project: backend.ProjectContext{Language: backend.LanguageAuto}, path: "main.go", want: true},
		{name: "auto Bash file", project: backend.ProjectContext{Language: backend.LanguageAuto}, path: "script.sh", want: false},
		{name: "explicit Bash", project: backend.ProjectContext{Language: backend.LanguageBash}, path: "script.sh", want: false},
		{name: "auto Makefile", project: backend.ProjectContext{Language: backend.LanguageAuto}, path: "Makefile", want: false},
		{name: "explicit Kotlin", project: backend.ProjectContext{Language: backend.LanguageKotlin}, path: "Main.kt", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldRunGofmt(test.project, test.path); got != test.want {
				t.Fatalf("shouldRunGofmt(%q)=%v, want %v", test.path, got, test.want)
			}
		})
	}
}

// WHY: This file verifies incremental diagnostic refresh after ordinary import edits without trusting stale package or sibling findings.

package godiagnostics_test

import "testing"

func TestManagedDiagnosticsImportChangesRefreshAffectedClosures(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{
		"provider/value.go":   "package provider\nfunc Value() int { return 1 }\n",
		"consumer/value.go":   "package consumer\nimport \"example.com/cacheinputs/provider\"\nvar Value = provider.Value()\n",
		"transitive/value.go": "package transitive\nimport \"example.com/cacheinputs/consumer\"\nvar Value int = consumer.Value\n",
		"unrelated/value.go":  "package unrelated\nfunc Value() int { return 1 }\n",
	})
	assertCapture := func(want string, pulls, starts int) {
		t.Helper()
		metrics := checkCachedInputsMatchFull(t, fixture, want)
		if got := metrics.Phases["gopls.request.textDocument/diagnostic"].Count; got != pulls {
			t.Errorf("diagnostic pulls=%d, want %d", got, pulls)
		}
		if got := metrics.Phases["gopls.request.initialize"].Count; got != starts {
			t.Errorf("gopls initialize requests=%d, want %d", got, starts)
		}
	}

	assertCapture("", 4, 1)
	assertCapture("", 0, 0)

	providerWithUnusedImport := "package provider\n\nimport \"fmt\"\n\nfunc Value() int { return 1 }\n"
	writeManagedFixture(t, fixture.root, "provider/value.go", providerWithUnusedImport)
	assertCapture("imported and not used", 3, 0)

	writeManagedFixture(t, fixture.root, "provider/value.go", "package provider\nfunc Value() int { return 1 }\n")
	assertCapture("", 3, 0)

	leafWithUnusedImport := "package transitive\n\nimport (\n\t\"fmt\"\n\t\"example.com/cacheinputs/consumer\"\n)\n\nvar Value int = consumer.Value\n"
	writeManagedFixture(t, fixture.root, "transitive/value.go", leafWithUnusedImport)
	assertCapture("imported and not used", 1, 0)

	writeManagedFixture(t, fixture.root, "transitive/value.go", "package transitive\nimport \"example.com/cacheinputs/consumer\"\nvar Value int = consumer.Value\n")
	assertCapture("", 1, 0)

	writeManagedFixture(t, fixture.root, "provider/value.go", providerWithUnusedImport)
	writeManagedFixture(t, fixture.root, "unrelated/value.go", "package unrelated\nfunc Value() int { return unrelatedMissing }\n")
	assertCapture("unrelatedMissing", 4, 0)
	assertCapture("imported and not used", 0, 0)

	writeManagedFixture(t, fixture.root, "provider/value.go", "package provider\n\nimport \"strings\"\n\nfunc Value() string { return strings.TrimSpace(\"changed\") }\n")
	assertCapture("cannot use", 3, 0)
	assertCapture("unrelatedMissing", 0, 0)

	writeManagedFixture(t, fixture.root, "provider/value.go", "package provider\nfunc Value() int { return 1 }\n")
	assertCapture("unrelatedMissing", 3, 0)
}

func TestManagedDiagnosticsEmbedBytesInvalidateAllCachedPackages(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{
		"embedded/value.go":    "package embedded\n\nimport _ \"embed\"\n\n//go:embed content.txt\nvar content string\n\nfunc Value() int { return len(content) }\n",
		"unrelated/value.go":   "package unrelated\nfunc Value() int { return 1 }\n",
		"embedded/content.txt": "first embedded value",
	})
	checkCachedInputsMatchFull(t, fixture, "")
	warm := checkCachedInputsMatchFull(t, fixture, "")
	if got := warm.Phases["gopls.request.textDocument/diagnostic"].Count; got != 0 {
		t.Fatalf("warm capture pulled %d files, want cached diagnostics", got)
	}

	writeManagedFixture(t, fixture.root, "embedded/content.txt", "changed embedded value")
	changed := checkCachedInputsMatchFull(t, fixture, "")
	if got := changed.Phases["gopls.request.textDocument/diagnostic"].Count; got != 2 {
		t.Fatalf("embed boundary change pulled %d files, want both cached packages", got)
	}
}

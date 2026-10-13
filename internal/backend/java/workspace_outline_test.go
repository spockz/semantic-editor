// WHY: Verifies Maven reactor discovery, Java package grouping, synopsis precedence, and test-source filtering.
package java

import (
	"os"
	"path/filepath"
	"testing"

	"semedit/internal/backend"
)

func TestWorkspaceOutlineDiscoversMavenModulesAndPackages(t *testing.T) {
	root := t.TempDir()
	writeJavaOutlineFixture(t, root, "pom.xml", `<project><groupId>com.acme</groupId><artifactId>acme-parent</artifactId><version>1.0.0</version><packaging>pom</packaging><modules><module>core</module><module>api</module></modules></project>`)
	writeJavaOutlineFixture(t, root, "core/pom.xml", `<project><parent><groupId>com.acme</groupId><version>1.0.0</version></parent><artifactId>core</artifactId></project>`)
	writeJavaOutlineFixture(t, root, "api/pom.xml", `<project><parent><groupId>com.acme</groupId><version>1.0.0</version></parent><artifactId>api</artifactId></project>`)
	writeJavaOutlineFixture(t, root, "core/src/main/java/com/acme/core/package-info.java", `/** Core domain model and validation. */
package com.acme.core;
`)
	writeJavaOutlineFixture(t, root, "core/src/main/java/com/acme/core/Money.java", `package com.acme.core;
/** A monetary amount. */
public record Money(long cents) {}
`)
	writeJavaOutlineFixture(t, root, "core/src/test/java/com/acme/core/tests/MoneyTest.java", `package com.acme.core.tests;
class MoneyTest {}
`)
	writeJavaOutlineFixture(t, root, "api/src/main/java/com/acme/api/Ledger.java", `package com.acme.api;
/** Public ledger service. */
public interface Ledger {}
`)

	result, err := WorkspaceOutline(root, ".", false)
	if err != nil {
		t.Fatalf("WorkspaceOutline: %v", err)
	}
	if len(result.Modules) != 3 || result.Modules[0].Path != "." || result.Modules[0].GroupID != "com.acme" || result.Modules[0].ArtifactID != "acme-parent" || result.Modules[0].Packaging != "pom" {
		t.Fatalf("Maven modules = %#v", result.Modules)
	}
	if len(result.Packages) != 2 {
		t.Fatalf("main-source packages = %#v", result.Packages)
	}
	core := findJavaWorkspacePackage(t, result.Packages, "core/com/acme/core")
	if core.Name != "com.acme.core" || core.FilesCount != 2 || core.Synopsis != "Core domain model and validation." || core.Doc == nil {
		t.Fatalf("core package = %#v", core)
	}
	api := findJavaWorkspacePackage(t, result.Packages, "api/com/acme/api")
	if api.Synopsis != "Public ledger service." {
		t.Fatalf("primary-type synopsis = %#v", api)
	}

	withTests, err := WorkspaceOutline(root, "core", true)
	if err != nil {
		t.Fatalf("WorkspaceOutline with tests: %v", err)
	}
	if len(withTests.Modules) != 2 || len(withTests.Packages) != 2 || withTests.Packages[0].FilesCount != 2 || withTests.Packages[1].Name != "com.acme.core.tests" {
		t.Fatalf("scoped test-source outline = %#v", withTests)
	}
	looksJava, err := LooksLikeJavaWorkspace(root, ".")
	if err != nil || !looksJava {
		t.Fatalf("LooksLikeJavaWorkspace() = %t, %v", looksJava, err)
	}
}

func TestWorkspaceOutlineRejectsPathsOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if _, err := WorkspaceOutline(root, outside, false); err == nil {
		t.Fatal("WorkspaceOutline accepted a path outside the root")
	}
}

func writeJavaOutlineFixture(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", relative, err)
	}
}

func findJavaWorkspacePackage(t *testing.T, packages []backend.WorkspacePackage, path string) backend.WorkspacePackage {
	t.Helper()
	for _, pkg := range packages {
		if pkg.Path == path {
			return pkg
		}
	}
	t.Fatalf("workspace package %q not found in %#v", path, packages)
	return backend.WorkspacePackage{}
}

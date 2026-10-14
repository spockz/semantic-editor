// WHY: Verifies Maven reactor discovery, Java package grouping, synopsis precedence, and test-source filtering.
package java

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/backend"
)

func TestWorkspaceOutlineDiscoversMavenModulesAndPackages(t *testing.T) {
	root := t.TempDir()
	writeJavaOutlineFixture(t, root, "pom.xml", `<project><groupId>com.acme</groupId><artifactId>acme-parent</artifactId><version>1.0.0</version><name>Acme workspace</name><description>Parent build
for Acme modules.</description><packaging>pom</packaging><modules><module>core</module><module>api</module></modules></project>`)
	writeJavaOutlineFixture(t, root, "core/pom.xml", `<project><parent><groupId>com.acme</groupId><version>1.0.0</version></parent><artifactId>core</artifactId><name>Core module</name><description>Shared core domain.</description></project>`)
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
	if result.Modules[0].Language != string(backend.LanguageJava) || result.Modules[0].Synopsis != "Parent build for Acme modules." || result.Modules[0].Doc == nil || *result.Modules[0].Doc != "Parent build\nfor Acme modules." || result.Modules[0].PackagesCount != 0 {
		t.Fatalf("root module metadata = %#v", result.Modules[0])
	}
	coreModule := findJavaWorkspaceModule(t, result.Modules, "core")
	if coreModule.Language != string(backend.LanguageJava) || coreModule.Synopsis != "Shared core domain." || coreModule.PackagesCount != 1 {
		t.Fatalf("core module metadata = %#v", coreModule)
	}
	if len(result.Packages) != 2 {
		t.Fatalf("main-source packages = %#v", result.Packages)
	}
	core := findJavaWorkspacePackage(t, result.Packages, "core/com/acme/core")
	if core.Name != "com.acme.core" || core.FilesCount != 2 || core.Synopsis != "Core domain model and validation." || core.Doc == nil {
		t.Fatalf("core package = %#v", core)
	}
	if core.Language != string(backend.LanguageJava) || core.Module != "com.acme:core" {
		t.Fatalf("core package ownership = %#v", core)
	}
	api := findJavaWorkspacePackage(t, result.Packages, "api/com/acme/api")
	if api.Synopsis != "Public ledger service." {
		t.Fatalf("primary-type synopsis = %#v", api)
	}

	withTests, err := WorkspaceOutline(root, "core", true)
	if err != nil {
		t.Fatalf("WorkspaceOutline with tests: %v", err)
	}
	if len(withTests.Modules) != 1 || withTests.Modules[0].Path != "core" || len(withTests.Packages) != 2 || withTests.Packages[0].FilesCount != 2 || withTests.Packages[1].Name != "com.acme.core.tests" {
		t.Fatalf("scoped test-source outline = %#v", withTests)
	}
	sourceScope, err := WorkspaceOutline(root, "core/src/main/java", false)
	if err != nil {
		t.Fatalf("WorkspaceOutline for module source subtree: %v", err)
	}
	if len(sourceScope.Modules) != 1 || sourceScope.Modules[0].Path != "core" || sourceScope.Modules[0].PackagesCount != 1 || len(sourceScope.Packages) != 1 || sourceScope.Packages[0].Module != "com.acme:core" {
		t.Fatalf("module source subtree outline = %#v", sourceScope)
	}
	looksJava, err := LooksLikeJavaWorkspace(root, ".")
	if err != nil || !looksJava {
		t.Fatalf("LooksLikeJavaWorkspace() = %t, %v", looksJava, err)
	}
}

func TestWorkspaceOutlinePropagatesMalformedMavenManifest(t *testing.T) {
	root := t.TempDir()
	writeJavaOutlineFixture(t, root, "java/pom.xml", `<project>`)
	_, err := WorkspaceOutline(root, "java", false)
	if err == nil || !strings.Contains(err.Error(), "parse Maven module POM") {
		t.Fatalf("WorkspaceOutline error = %v, want malformed POM parse error", err)
	}
}

func TestWorkspaceOutlineScopedMavenDiscoverySkipsMalformedSibling(t *testing.T) {
	root := t.TempDir()
	writeJavaOutlineFixture(t, root, "pom.xml", `<project><groupId>com.acme</groupId><artifactId>parent</artifactId><modules><module>core</module><module>api</module></modules></project>`)
	writeJavaOutlineFixture(t, root, "core/pom.xml", `<project><artifactId>core</artifactId></project>`)
	writeJavaOutlineFixture(t, root, "core/src/main/java/com/acme/Core.java", "package com.acme; class Core {}")
	writeJavaOutlineFixture(t, root, "api/pom.xml", `<project>`)

	result, err := WorkspaceOutline(root, "core", false)
	if err != nil {
		t.Fatalf("scoped WorkspaceOutline: %v", err)
	}
	if len(result.Modules) != 1 || result.Modules[0].Path != "core" || result.Modules[0].PackagesCount != 1 {
		t.Fatalf("scoped modules = %#v; want core with one package", result.Modules)
	}
	if len(result.Packages) != 1 || result.Packages[0].Module != "com.acme:core" {
		t.Fatalf("scoped package ownership = %#v; want com.acme:core", result.Packages)
	}
	if _, err := WorkspaceOutline(root, ".", false); err == nil || !strings.Contains(err.Error(), "parse Maven module POM") {
		t.Fatalf("unscoped WorkspaceOutline error = %v; want malformed sibling error", err)
	}
}

func TestWorkspaceOutlineDiscoversIndependentMavenModuleOwnership(t *testing.T) {
	root := t.TempDir()
	writeJavaOutlineFixture(t, root, "pom.xml", `<project><groupId>com.acme</groupId><artifactId>reactor</artifactId><modules><module>core</module></modules></project>`)
	writeJavaOutlineFixture(t, root, "core/pom.xml", `<project><artifactId>core</artifactId></project>`)
	writeJavaOutlineFixture(t, root, "independent/pom.xml", `<project><groupId>org.independent</groupId><artifactId>standalone</artifactId><version>2.0</version></project>`)
	writeJavaOutlineFixture(t, root, "independent/src/main/java/org/example/Standalone.java", "package org.example; class Standalone {}")

	scoped, err := WorkspaceOutline(root, "independent", false)
	if err != nil {
		t.Fatalf("scoped WorkspaceOutline: %v", err)
	}
	if len(scoped.Modules) != 1 || scoped.Modules[0].Path != "independent" || scoped.Modules[0].PackagesCount != 1 {
		t.Fatalf("scoped modules = %#v; want only independent with one package", scoped.Modules)
	}
	if len(scoped.Packages) != 1 || scoped.Packages[0].Module != "org.independent:standalone" {
		t.Fatalf("scoped package ownership = %#v; want independent coordinates", scoped.Packages)
	}

	unscoped, err := WorkspaceOutline(root, ".", false)
	if err != nil {
		t.Fatalf("unscoped WorkspaceOutline: %v", err)
	}
	independent := findJavaWorkspaceModule(t, unscoped.Modules, "independent")
	if independent.PackagesCount != 1 {
		t.Fatalf("unscoped independent module = %#v; want one package", independent)
	}
	packageFound := false
	for _, pkg := range unscoped.Packages {
		if pkg.Path == "independent/org/example" {
			packageFound = true
			if pkg.Module != "org.independent:standalone" {
				t.Fatalf("unscoped independent package ownership = %#v", pkg)
			}
		}
	}
	if !packageFound {
		t.Fatal("unscoped outline omitted independent package")
	}
}

func TestLooksLikeJavaWorkspaceRequiresRootEvidenceUnlessScoped(t *testing.T) {
	root := t.TempDir()
	writeJavaOutlineFixture(t, root, "docs/examples/Sample.java", "package sample; class Sample {}")
	looks, err := LooksLikeJavaWorkspace(root, ".")
	if err != nil || looks {
		t.Fatalf("unscoped Java sample detection = %t, %v; want false", looks, err)
	}
	looks, err = LooksLikeJavaWorkspace(root, "docs/examples")
	if err != nil || !looks {
		t.Fatalf("scoped Java sample detection = %t, %v; want true", looks, err)
	}
	writeJavaOutlineFixture(t, root, "nested/pom.xml", "<project/>")
	looks, err = LooksLikeJavaWorkspace(root, ".")
	if err != nil || !looks {
		t.Fatalf("nested manifest detection = %t, %v; want true", looks, err)
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

func findJavaWorkspaceModule(t *testing.T, modules []backend.WorkspaceModule, path string) backend.WorkspaceModule {
	t.Helper()
	for _, module := range modules {
		if module.Path == path {
			return module
		}
	}
	t.Fatalf("Maven module %q not found in %#v", path, modules)
	return backend.WorkspaceModule{}
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

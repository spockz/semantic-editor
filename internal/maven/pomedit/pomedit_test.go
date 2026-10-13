// WHY: Verifies Maven reactor dependency cases while asserting comments and existing XML text remain unchanged.
package pomedit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanAlreadyManagedDependency(t *testing.T) {
	root := t.TempDir()
	writePOM(t, filepath.Join(root, "pom.xml"), `<project>
  <groupId>acme</groupId>
  <artifactId>parent</artifactId>
  <version>1</version>
  <modules><module>child</module></modules>
  <dependencyManagement>
    <!-- pinned in the parent -->
    <dependencies>
      <dependency><groupId>acme.libs</groupId><artifactId>core</artifactId><version>2.4</version></dependency>
    </dependencies>
  </dependencyManagement>
</project>
`)
	child := filepath.Join(root, "child", "pom.xml")
	writePOM(t, child, `<project>
  <parent><groupId>acme</groupId><artifactId>parent</artifactId><version>1</version></parent>
  <artifactId>child</artifactId>
  <dependencies>
  </dependencies>
</project>
`)
	edits, err := Plan(root, child, Dependency{GroupID: "acme.libs", ArtifactID: "core"})
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) != 1 || edits[0].Path != canonicalPath(t, child) {
		t.Fatalf("edits = %#v, want only child POM", edits)
	}
	got := string(edits[0].Data)
	if !strings.Contains(got, "<groupId>acme.libs</groupId>") || strings.Contains(got, "<version>2.4</version>") {
		t.Fatalf("managed child edit = %s", got)
	}
}

func TestPlanRegistersUnmanagedDependencyInParent(t *testing.T) {
	root := t.TempDir()
	rootPOM := filepath.Join(root, "pom.xml")
	writePOM(t, rootPOM, `<project>
  <groupId>acme</groupId>
  <artifactId>parent</artifactId>
  <version>1</version>
  <modules><module>child</module></modules>
  <dependencyManagement>
    <!-- preserve this comment -->
    <dependencies></dependencies>
  </dependencyManagement>
</project>
`)
	child := filepath.Join(root, "child", "pom.xml")
	writePOM(t, child, `<project>
  <parent><groupId>acme</groupId><artifactId>parent</artifactId><version>1</version></parent>
  <artifactId>child</artifactId>
  <dependencies></dependencies>
</project>
`)
	edits, err := Plan(root, child, Dependency{GroupID: "org.example", ArtifactID: "codec", Version: "3.1", Scope: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) != 2 || edits[0].Path != canonicalPath(t, rootPOM) || edits[1].Path != canonicalPath(t, child) {
		t.Fatalf("edits = %#v, want parent then child", edits)
	}
	managed := string(edits[0].Data)
	childText := string(edits[1].Data)
	if !strings.Contains(managed, "<!-- preserve this comment -->") || !strings.Contains(managed, "<version>3.1</version>") {
		t.Fatalf("parent managed edit = %s", managed)
	}
	if !strings.Contains(childText, "<artifactId>codec</artifactId>") || strings.Contains(childText, "<version>3.1</version>") || !strings.Contains(childText, "<scope>test</scope>") {
		t.Fatalf("child dependency edit = %s", childText)
	}
}

func TestPlanStandalonePOMUsesFullCoordinate(t *testing.T) {
	root := t.TempDir()
	pom := filepath.Join(root, "pom.xml")
	writePOM(t, pom, `<project>
  <modelVersion>4.0.0</modelVersion>
  <artifactId>standalone</artifactId>
</project>
`)
	edits, err := Plan(root, "", Dependency{GroupID: "org.example", ArtifactID: "codec", Version: "3.1", Scope: "runtime", Classifier: "jdk"})
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) != 1 || edits[0].Path != canonicalPath(t, pom) {
		t.Fatalf("edits = %#v, want standalone POM", edits)
	}
	got := string(edits[0].Data)
	for _, expected := range []string{"<groupId>org.example</groupId>", "<artifactId>codec</artifactId>", "<version>3.1</version>", "<scope>runtime</scope>", "<classifier>jdk</classifier>"} {
		if !strings.Contains(got, expected) {
			t.Errorf("standalone POM missing %q: %s", expected, got)
		}
	}
}

func TestParseCoordinate(t *testing.T) {
	got, err := ParseCoordinate("org.example:codec:3.1:test:tests")
	if err != nil {
		t.Fatal(err)
	}
	if got != (Dependency{GroupID: "org.example", ArtifactID: "codec", Version: "3.1", Scope: "test", Classifier: "tests"}) {
		t.Fatalf("coordinate = %#v", got)
	}
	for _, invalid := range []string{"", "g", "g:a:x:y:z:w", "bad group:a:1", "g:a:1:unknown"} {
		if _, err := ParseCoordinate(invalid); err == nil {
			t.Errorf("coordinate %q accepted", invalid)
		}
	}
}

func writePOM(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func canonicalPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// Package java_test integration-tests behaviors that a fake LSP session cannot prove.
package java_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"semedit/internal/backend"
	javabackend "semedit/internal/backend/java"
	"semedit/internal/maven/pomedit"
	"semedit/internal/operation"
	"semedit/internal/pipeline"
)

//nolint:gocognit,funlen // The end-to-end scenario intentionally exercises one real server session across the Java operation lifecycle.
func TestRealJavaJDTLSIntegration(t *testing.T) {
	jdtlsHome := os.Getenv("SEMEDIT_REAL_JDTLS_HOME")
	javaHome := os.Getenv("SEMEDIT_REAL_JAVA_HOME")
	if jdtlsHome == "" || javaHome == "" {
		t.Skip("set SEMEDIT_REAL_JDTLS_HOME and SEMEDIT_REAL_JAVA_HOME to run real JDT LS integration")
	}
	var err error
	jdtlsHome, err = filepath.Abs(jdtlsHome)
	if err != nil {
		t.Fatalf("resolve JDT LS home: %v", err)
	}
	if _, statErr := os.Stat(jdtlsHome); statErr != nil {
		t.Fatalf("JDT LS home: %v", statErr)
	}
	javaHome, err = filepath.Abs(javaHome)
	if err != nil {
		t.Fatalf("resolve Java home: %v", err)
	}
	javaBin := filepath.Join(javaHome, "bin", "java")
	if _, statErr := os.Stat(javaBin); statErr != nil {
		t.Fatalf("Java runtime: %v", statErr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if runtimeErr := validateRealJavaRuntime(ctx, javaBin); runtimeErr != nil {
		t.Fatalf("require JDK 21 or newer: %v", runtimeErr)
	}
	t.Setenv("JAVA_HOME", javaHome)

	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	scratch := filepath.Join(repoRoot, ".scratch")
	if mkdirErr := os.MkdirAll(scratch, 0o700); mkdirErr != nil {
		t.Fatalf("create scratch directory: %v", mkdirErr)
	}
	root, err := os.MkdirTemp(scratch, "java-real-integration-")
	if err != nil {
		t.Fatalf("create Maven reactor fixture: %v", err)
	}
	t.Cleanup(func() {
		if os.Getenv("SEMEDIT_REAL_KEEP_FIXTURE") == "1" {
			t.Logf("preserved real Java fixture at %s", root)
			return
		}
		if removeErr := os.RemoveAll(root); removeErr != nil {
			t.Errorf("remove Java integration fixture: %v", removeErr)
		}
	})

	fixture := map[string]string{
		"pom.xml":                `<project xmlns="http://maven.apache.org/POM/4.0.0"><modelVersion>4.0.0</modelVersion><groupId>com.acme.ledger</groupId><artifactId>acme-ledger-parent</artifactId><version>1.0.0-SNAPSHOT</version><packaging>pom</packaging><properties><maven.compiler.release>21</maven.compiler.release><project.build.sourceEncoding>UTF-8</project.build.sourceEncoding></properties><modules><module>ledger-domain</module><module>ledger-service</module></modules><dependencyManagement><dependencies><dependency><groupId>com.acme.ledger</groupId><artifactId>ledger-domain</artifactId><version>${project.version}</version></dependency><dependency><groupId>org.slf4j</groupId><artifactId>slf4j-api</artifactId><version>2.0.12</version></dependency></dependencies></dependencyManagement></project>`,
		"ledger-domain/pom.xml":  `<project xmlns="http://maven.apache.org/POM/4.0.0"><modelVersion>4.0.0</modelVersion><parent><groupId>com.acme.ledger</groupId><artifactId>acme-ledger-parent</artifactId><version>1.0.0-SNAPSHOT</version><relativePath>../pom.xml</relativePath></parent><artifactId>ledger-domain</artifactId></project>`,
		"ledger-service/pom.xml": `<project xmlns="http://maven.apache.org/POM/4.0.0"><modelVersion>4.0.0</modelVersion><parent><groupId>com.acme.ledger</groupId><artifactId>acme-ledger-parent</artifactId><version>1.0.0-SNAPSHOT</version><relativePath>../pom.xml</relativePath></parent><artifactId>ledger-service</artifactId><dependencies><dependency><groupId>com.acme.ledger</groupId><artifactId>ledger-domain</artifactId></dependency></dependencies></project>`,
		"ledger-domain/src/main/java/com/acme/ledger/domain/Money.java": `package com.acme.ledger.domain;

public record Money(long cents) {}
`,
		"ledger-domain/src/main/java/com/acme/ledger/domain/EntryType.java": `package com.acme.ledger.domain;

public enum EntryType { DEBIT, CREDIT }
`,
		"ledger-domain/src/main/java/com/acme/ledger/domain/LedgerEntry.java": `package com.acme.ledger.domain;

public record LedgerEntry(Money amount, EntryType type) {}
`,
		"ledger-domain/src/main/java/com/acme/ledger/domain/package-info.java": `/** Core immutable ledger domain models and contracts. */
package com.acme.ledger.domain;
`,
		"ledger-service/src/main/java/com/acme/ledger/service/package-info.java": `/** Transactional ledger posting and balance services. */
package com.acme.ledger.service;
`,
		"ledger-domain/src/main/java/com/acme/ledger/domain/AuditSink.java": `package com.acme.ledger.domain;

public interface AuditSink {
    void record(LedgerEntry entry);
}
`,
		"ledger-service/src/main/java/com/acme/ledger/service/LedgerService.java": `package com.acme.ledger.service;

import com.acme.ledger.domain.AuditSink;
import com.acme.ledger.domain.EntryType;
import com.acme.ledger.domain.LedgerEntry;
import com.acme.ledger.domain.Money;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

public final class LedgerService {
    private final AuditSink auditSink;
    private final List<LedgerEntry> entries = new ArrayList<>();
    private final Map<String, Integer> limits = Map.of("DAILY", 1000);

    public LedgerService(AuditSink auditSink) {
        this.auditSink = auditSink;
    }

    void post(LedgerEntry entry) {
        post(entry, true);
    }

    void post(LedgerEntry entry, boolean notifyAudit) {
        long cents = entry.amount().cents();
        if (cents <= 0) {
            throw new IllegalArgumentException("amount must be positive");
        }
        if (cents > 1000) {
            throw new IllegalArgumentException("amount exceeds posting limit");
        }
        entries.add(entry);
        if (notifyAudit) {
            auditSink.record(entry);
        }
    }

    public long netBalanceCents() {
        long total = 0L;
        for (LedgerEntry entry : entries) {
            long signed = switch (entry.type()) {
                case CREDIT -> entry.amount().cents();
                case DEBIT -> -entry.amount().cents();
            };
            total += signed;
        }
        return total;
    }

    int configuredDailyLimit() {
        return limits.get("DAILY");
    }
}
`,
		"ledger-service/src/main/java/com/acme/ledger/service/LedgerCaller.java": `package com.acme.ledger.service;

import com.acme.ledger.domain.EntryType;
import com.acme.ledger.domain.LedgerEntry;
import com.acme.ledger.domain.Money;

final class LedgerCaller {
    void submit(LedgerService service) { service.post(new LedgerEntry(new Money(100), EntryType.CREDIT), true); }
}
`,
		"ledger-service/src/test/java/com/acme/ledger/service/LedgerServiceTest.java": `package com.acme.ledger.service;

import com.acme.ledger.domain.EntryType;
import com.acme.ledger.domain.LedgerEntry;
import com.acme.ledger.domain.Money;

final class LedgerServiceTest {
    void testPost(LedgerService service) { service.post(new LedgerEntry(new Money(2), EntryType.DEBIT), false); }
}
`,
		"ledger-service/src/main/java/com/acme/ledger/service/InMemoryAuditSink.java": `package com.acme.ledger.service;

public final class InMemoryAuditSink {}
`,
	}
	for path, contents := range fixture {
		file := filepath.Join(root, filepath.FromSlash(path))
		if mkdirErr := os.MkdirAll(filepath.Dir(file), 0o700); mkdirErr != nil {
			t.Fatalf("create fixture directory for %s: %v", path, mkdirErr)
		}
		if writeErr := pipeline.WriteAtomic(file, []byte(contents)); writeErr != nil {
			t.Fatalf("write fixture %s: %v", path, writeErr)
		}
	}

	workspaceOutline, err := javabackend.WorkspaceOutline(root, ".", true)
	if err != nil {
		t.Fatalf("real Maven reactor workspace outline: %v", err)
	}
	if len(workspaceOutline.Modules) != 3 {
		t.Fatalf("real Maven reactor modules = %#v", workspaceOutline.Modules)
	}
	wantModules := map[string]string{".": "acme-ledger-parent", "ledger-domain": "ledger-domain", "ledger-service": "ledger-service"}
	for _, module := range workspaceOutline.Modules {
		if module.ArtifactID != wantModules[module.Path] || module.GroupID != "com.acme.ledger" || module.Version != "1.0.0-SNAPSHOT" {
			t.Fatalf("unexpected Maven module metadata: %#v", module)
		}
		delete(wantModules, module.Path)
	}
	if len(wantModules) != 0 {
		t.Fatalf("workspace outline omitted Maven modules: %#v", wantModules)
	}
	if _, coordinateErr := pomedit.ParseCoordinate("org.slf4j:slf4j-api:2.0.12"); coordinateErr != nil {
		t.Fatalf("parse managed Maven dependency coordinate: %v", coordinateErr)
	}
	if len(workspaceOutline.Packages) != 2 {
		t.Fatalf("real Maven reactor package grouping = %#v", workspaceOutline.Packages)
	}
	packageSummaries := make(map[string]string, len(workspaceOutline.Packages))
	packageFileCounts := make(map[string]int, len(workspaceOutline.Packages))
	for _, pkg := range workspaceOutline.Packages {
		packageSummaries[pkg.Name] = pkg.Synopsis
		packageFileCounts[pkg.Name] = pkg.FilesCount
	}
	if packageSummaries["com.acme.ledger.domain"] != "Core immutable ledger domain models and contracts." || packageSummaries["com.acme.ledger.service"] != "Transactional ledger posting and balance services." {
		t.Fatalf("workspace outline package synopses = %#v", packageSummaries)
	}
	if packageFileCounts["com.acme.ledger.service"] != 5 {
		t.Fatalf("workspace outline omitted main or test Java sources: %#v", packageFileCounts)
	}
	withoutTests, err := javabackend.WorkspaceOutline(root, ".", false)
	if err != nil {
		t.Fatalf("real Maven reactor workspace outline without tests: %v", err)
	}
	for _, pkg := range withoutTests.Packages {
		if pkg.Name == "com.acme.ledger.service" && pkg.FilesCount != 4 {
			t.Fatalf("workspace outline included test sources without include_tests: %#v", pkg)
		}
	}
	contractTarget := filepath.Join(root, "ledger-domain", "src", "main", "java", "com", "acme", "ledger", "domain", "contracts", "ContractTarget.java")
	if mkdirErr := os.MkdirAll(filepath.Dir(contractTarget), 0o700); mkdirErr != nil {
		t.Fatalf("create Java move destination package: %v", mkdirErr)
	}
	if writeErr := pipeline.WriteAtomic(contractTarget, []byte("package com.acme.ledger.domain.contracts;\n\npublic final class ContractTarget {}\n")); writeErr != nil {
		t.Fatalf("write Java move destination type: %v", writeErr)
	}
	servicePOM := filepath.Join(root, "ledger-service", "pom.xml")
	managed, err := pomedit.Plan(root, servicePOM, pomedit.Dependency{GroupID: "org.slf4j", ArtifactID: "slf4j-api", Version: "2.0.12"})
	if err != nil || len(managed) != 1 || bytes.Contains(managed[0].Data, []byte("<version>2.0.12</version>")) {
		t.Fatalf("plan managed reactor dependency: edits=%#v err=%v", managed, err)
	}
	unmanaged, err := pomedit.Plan(root, servicePOM, pomedit.Dependency{GroupID: "com.google.guava", ArtifactID: "guava", Version: "33.0.0-jre"})
	if err != nil || len(unmanaged) != 2 {
		t.Fatalf("plan new reactor dependency: edits=%#v err=%v", unmanaged, err)
	}
	for _, edit := range unmanaged {
		if edit.Path == filepath.Join(root, "pom.xml") && !bytes.Contains(edit.Data, []byte("<version>33.0.0-jre</version>")) {
			t.Fatalf("root dependency management omitted Guava version: %s", edit.Data)
		}
		if edit.Path == servicePOM && bytes.Contains(edit.Data, []byte("<version>33.0.0-jre</version>")) {
			t.Fatalf("service POM duplicated the managed Guava version: %s", edit.Data)
		}
	}

	serviceFile := filepath.Join(root, "ledger-service", "src", "main", "java", "com", "acme", "ledger", "service", "LedgerService.java")
	project := backend.ProjectContext{
		RootDir:        root,
		File:           serviceFile,
		Language:       backend.LanguageJava,
		WorkspaceTrust: backend.NewWorkspaceTrust(root, true),
		Java:           backend.JavaConfig{JDTLSHome: jdtlsHome, JavaBin: javaBin, ImportMaven: true, ImportMavenSet: true},
	}
	service := javabackend.NewJavaBackend()
	t.Cleanup(func() {
		if closeErr := service.Close(); closeErr != nil {
			t.Errorf("close JDT LS session: %v", closeErr)
		}
	})
	backendRegistry, err := backend.NewRegistry(service)
	if err != nil {
		t.Fatalf("register Java backend for semantic operations: %v", err)
	}
	operationRegistry := operation.DefaultRegistry()
	call := operation.NewCallContext(root, project)
	call.Ctx, call.Service, call.Registry = ctx, backend.NewService(backendRegistry), operationRegistry
	outlineResult, err := operationRegistry.Dispatch(call, "workspace_outline", map[string]any{
		"path": ".", "language": "java", "include_tests": true,
	})
	if err != nil {
		t.Fatalf("real semantic_workspace_outline Java: %v", err)
	}
	if result, ok := outlineResult.(operation.WorkspaceOutlineRes); !ok || result.Result == nil || len(result.Result.Modules) != 3 {
		t.Fatalf("real Java workspace-outline operation result = %#v", outlineResult)
	}
	insertResult, err := operationRegistry.Dispatch(call, "insert_construct", map[string]any{
		"file":     filepath.ToSlash(filepath.Join("ledger-service", "src/main/java/com/acme/ledger/service/LedgerService.java")),
		"language": "java", "trust_workspace": true, "jdtls_home": jdtlsHome, "java_bin": javaBin, "import_maven": true,
		"kind": "element", "in_declaration": "LedgerService.limits", "discriminator": `Map.of("DAILY", 1000)`,
		"source": `"MONTHLY", 5000`,
	})
	if err != nil {
		t.Fatalf("real semantic_insert_construct Java element: %v", err)
	}
	if _, ok := insertResult.(operation.FileEditRes); !ok {
		t.Fatalf("real Java element insertion result has type %T", insertResult)
	}
	serviceSource, err := os.ReadFile(serviceFile)
	if err != nil || !bytes.Contains(serviceSource, []byte(`Map.of("DAILY", 1000, "MONTHLY", 5000)`)) {
		t.Fatalf("real semantic_insert_construct did not append the Map.of entry: source=%s err=%v", serviceSource, err)
	}

	outline, err := service.Outline(ctx, backend.OutlineRequest{Project: project, Path: serviceFile, IncludeUnexported: true})
	if err != nil {
		t.Fatalf("real JDT LS outline: %v", err)
	}
	if outline == nil || len(outline.Files) != 1 || len(outline.Files[0].Symbols) == 0 {
		t.Fatalf("real JDT LS outline returned no declarations: %#v", outline)
	}

	selected, err := service.Lookup(ctx, project, "LedgerService.post(LedgerEntry, boolean)")
	if err != nil {
		t.Fatalf("real JDT LS overload lookup: %v", err)
	}
	if selected == nil || selected.Definition == nil || selected.Definition.QualifiedName != "LedgerService.post(LedgerEntry, boolean)" {
		t.Fatalf("real JDT LS overload selection = %#v", selected)
	}
	ambiguous, err := service.Lookup(ctx, project, "LedgerService.post")
	if err != nil {
		t.Fatalf("real JDT LS overload ambiguity: %v", err)
	}
	if ambiguous == nil || !ambiguous.IsAmbiguous() || len(ambiguous.Candidates) != 2 || ambiguous.Candidates[0].QualifiedName != "LedgerService.post(LedgerEntry)" || ambiguous.Candidates[1].QualifiedName != "LedgerService.post(LedgerEntry, boolean)" {
		t.Fatalf("real JDT LS overload candidates = %#v", ambiguous)
	}
	lookupResult, err := operationRegistry.Dispatch(call, "lookup", map[string]any{
		"file":   filepath.ToSlash(filepath.Join("ledger-service", "src/main/java/com/acme/ledger/service/LedgerService.java")),
		"symbol": "LedgerService.post", "language": "java", "trust_workspace": true,
		"jdtls_home": jdtlsHome, "java_bin": javaBin, "import_maven": true,
	})
	if err != nil {
		t.Fatalf("real semantic_lookup Java overloads: %v", err)
	}
	lookupOperationResult, lookupOK := lookupResult.(*operation.LookupRes)
	if !lookupOK || lookupOperationResult == nil || !lookupOperationResult.Ambiguous || len(lookupOperationResult.Candidates) != 2 || lookupOperationResult.Candidates[0].QualifiedName != "LedgerService.post(LedgerEntry)" || lookupOperationResult.Candidates[1].QualifiedName != "LedgerService.post(LedgerEntry, boolean)" {
		t.Fatalf("real semantic_lookup operation candidates = %#v", lookupResult)
	}
	for _, caller := range []string{
		filepath.Join("ledger-service", "src/main/java/com/acme/ledger/service/LedgerCaller.java"),
		filepath.Join("ledger-service", "src/test/java/com/acme/ledger/service/LedgerServiceTest.java"),
	} {
		if _, outlineErr := service.Outline(ctx, backend.OutlineRequest{Project: project, Path: filepath.Join(root, caller), IncludeUnexported: true}); outlineErr != nil {
			t.Fatalf("open real Java reference caller %s: %v", caller, outlineErr)
		}
	}

	references, err := service.FindReferences(ctx, backend.ReferencesRequest{Project: project, Symbol: "LedgerService.post(LedgerEntry, boolean)"})
	if err != nil {
		t.Fatalf("real JDT LS references: %v", err)
	}
	if references == nil || len(references.References) == 0 {
		t.Fatalf("real JDT LS returned no caller references: %#v", references)
	}
	callerFiles := make(map[string]bool, len(references.References))
	for _, reference := range references.References {
		callerFiles[filepath.Base(reference.File)] = true
	}
	if !callerFiles["LedgerCaller.java"] || !callerFiles["LedgerServiceTest.java"] {
		t.Fatalf("real JDT LS references omit main or test call sites: %#v", callerFiles)
	}
	operationReferences, err := operationRegistry.Dispatch(call, "find_references", map[string]any{
		"file":   filepath.ToSlash(filepath.Join("ledger-domain", "src/main/java/com/acme/ledger/domain/Money.java")),
		"symbol": "Money.cents", "language": "java", "trust_workspace": true,
		"jdtls_home": jdtlsHome, "java_bin": javaBin, "import_maven": true,
	})
	if err != nil {
		t.Fatalf("real semantic_find_references Java: %v", err)
	}
	if result, ok := operationReferences.(*backend.ReferencesResult); !ok || result == nil || len(result.References) == 0 {
		t.Fatalf("real Java references operation result = %#v", operationReferences)
	}
	before, err := os.ReadFile(serviceFile)
	if err != nil {
		t.Fatalf("read real refactoring preimage: %v", err)
	}
	selectedExpr := []byte("entry.amount().cents()")
	if !bytes.Contains(before, selectedExpr) {
		t.Fatalf("real refactoring expression is absent")
	}
	variableResult, err := operationRegistry.Dispatch(call, "extract_variable", map[string]any{
		"file":     filepath.ToSlash(filepath.Join("ledger-service", "src/main/java/com/acme/ledger/service/LedgerService.java")),
		"language": "java", "trust_workspace": true, "jdtls_home": jdtlsHome, "java_bin": javaBin, "import_maven": true,
		"symbol": "amountCents", "in_function": "LedgerService.post(LedgerEntry, boolean)", "discriminator": string(selectedExpr),
	})
	if err != nil {
		t.Fatalf("real semantic_extract_variable Java: %v", err)
	}
	if _, ok := variableResult.(operation.ExtractionResult); !ok {
		t.Fatalf("real Java variable extraction result has type %T", variableResult)
	}
	serviceSource, err = os.ReadFile(serviceFile)
	if err != nil || !strings.Contains(string(serviceSource), "long amountCents = entry.amount().cents();") || !strings.Contains(string(serviceSource), "long cents = amountCents;") {
		t.Fatalf("real semantic_extract_variable did not create and use amountCents: source=%s err=%v", serviceSource, err)
	}
	diagnostics, err := service.Verify(ctx, backend.VerifyRequest{Project: project, Path: serviceFile, CheckOnly: true})
	if err != nil {
		t.Fatalf("real JDT LS diagnostics: %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("real JDT LS diagnostics = %#v, want no diagnostics", diagnostics)
	}
	loopResult, err := operationRegistry.Dispatch(call, "inspect_construct", map[string]any{
		"file":     filepath.ToSlash(filepath.Join("ledger-service", "src/main/java/com/acme/ledger/service/LedgerService.java")),
		"language": "java", "trust_workspace": true, "jdtls_home": jdtlsHome, "java_bin": javaBin, "import_maven": true,
		"in_function": "LedgerService.netBalanceCents()", "kind": "loop",
	})
	if err != nil {
		t.Fatalf("real semantic_inspect_construct Java loop: %v", err)
	}
	if result, ok := loopResult.(operation.InspectConstructRes); !ok || len(result.Candidates) != 1 || result.Candidates[0].ConstructPath != "loop[0]" {
		t.Fatalf("real Java loop-inspection result = %#v", loopResult)
	}
	constantResult, err := operationRegistry.Dispatch(call, "extract_constant", map[string]any{
		"file":     filepath.ToSlash(filepath.Join("ledger-service", "src/main/java/com/acme/ledger/service/LedgerService.java")),
		"language": "java", "trust_workspace": true, "jdtls_home": jdtlsHome, "java_bin": javaBin, "import_maven": true,
		"symbol": "MAX_POSTING_CENTS", "in_function": "LedgerService.post(LedgerEntry, boolean)", "discriminator": "1000",
	})
	if err != nil {
		t.Fatalf("real semantic_extract_constant Java: %v", err)
	}
	if _, ok := constantResult.(operation.ExtractionResult); !ok {
		t.Fatalf("real Java constant extraction result has type %T", constantResult)
	}
	serviceSource, err = os.ReadFile(serviceFile)
	if err != nil || !bytes.Contains(serviceSource, []byte("MAX_POSTING_CENTS")) || !bytes.Contains(serviceSource, []byte("cents > MAX_POSTING_CENTS")) {
		t.Fatalf("real semantic_extract_constant did not create and use MAX_POSTING_CENTS: source=%s err=%v", serviceSource, err)
	}

	parameterObjectResult, err := operationRegistry.Dispatch(call, "introduce_parameter_object", map[string]any{
		"file":   filepath.ToSlash(filepath.Join("ledger-service", "src/main/java/com/acme/ledger/service/LedgerService.java")),
		"symbol": "LedgerService.post(LedgerEntry, boolean)", "destination_file": filepath.ToSlash(filepath.Join("ledger-service", "src/main/java/com/acme/ledger/service/LedgerService.java")),
		"parameter": map[string]any{"name": "options", "type": "PostOptions"},
		"fields": []any{
			map[string]any{"source_index": 0, "field": map[string]any{"name": "entry", "type": "LedgerEntry"}},
			map[string]any{"source_index": 1, "field": map[string]any{"name": "notifyAudit", "type": "boolean"}},
		},
		"language": "java", "trust_workspace": true, "jdtls_home": jdtlsHome, "java_bin": javaBin, "import_maven": true,
	})
	if err != nil {
		t.Fatalf("real semantic_introduce_parameter_object Java: %v", err)
	}
	if parameterObjectResult == nil {
		t.Fatal("real Java parameter-object operation returned nil")
	}
	serviceSource, err = os.ReadFile(serviceFile)
	if err != nil || !strings.Contains(string(serviceSource), "post(PostOptions options)") || !strings.Contains(string(serviceSource), "options.entry()") || !strings.Contains(string(serviceSource), "options.notifyAudit()") {
		t.Fatalf("real semantic_introduce_parameter_object did not update the method: source=%s err=%v", serviceSource, err)
	}
	callerAfter, err := os.ReadFile(filepath.Join(filepath.Dir(serviceFile), "LedgerCaller.java"))
	if err != nil || !strings.Contains(string(callerAfter), "new PostOptions(new LedgerEntry(new Money(100), EntryType.CREDIT), true)") {
		t.Fatalf("real semantic_introduce_parameter_object did not preserve caller arguments: source=%s err=%v", callerAfter, err)
	}
	testCallerAfter, err := os.ReadFile(filepath.Join(root, "ledger-service", "src", "test", "java", "com", "acme", "ledger", "service", "LedgerServiceTest.java"))
	if err != nil || !strings.Contains(string(testCallerAfter), "new PostOptions(new LedgerEntry(new Money(2), EntryType.DEBIT), false)") {
		t.Fatalf("real semantic_introduce_parameter_object did not preserve test caller arguments: source=%s err=%v", testCallerAfter, err)
	}

	implementationFile := filepath.Join(root, "ledger-service", "src", "main", "java", "com", "acme", "ledger", "service", "InMemoryAuditSink.java")
	implementationRelative := filepath.ToSlash(filepath.Join("ledger-service", "src/main/java/com/acme/ledger/service/InMemoryAuditSink.java"))
	_, err = operationRegistry.Dispatch(call, "implement_contract", map[string]any{
		"file": implementationRelative, "symbol": "InMemoryAuditSink", "contract": "AuditSink",
		"contract_file": filepath.ToSlash(filepath.Join("ledger-domain", "src/main/java/com/acme/ledger/domain/AuditSink.java")),
		"receiver_kind": "pointer", "body_policy": "panic_stub", "language": "java", "trust_workspace": true,
		"jdtls_home": jdtlsHome, "java_bin": javaBin, "import_maven": true,
	})
	if err != nil {
		t.Fatalf("real semantic_implement_contract Java: %v", err)
	}
	implementationSource, err := os.ReadFile(implementationFile)
	if err != nil || !strings.Contains(string(implementationSource), "@Override") || !strings.Contains(string(implementationSource), "record(LedgerEntry entry)") {
		t.Fatalf("real semantic_implement_contract did not generate the interface method: source=%s err=%v", implementationSource, err)
	}
	implementationProject := project
	implementationProject.File = implementationFile
	contractDiagnostics, err := service.Verify(ctx, backend.VerifyRequest{Project: implementationProject, Path: implementationFile, CheckOnly: true})
	if err != nil {
		t.Fatalf("verify implemented Java contract: %v", err)
	}
	if len(contractDiagnostics) != 0 {
		t.Fatalf("implemented Java contract diagnostics = %#v, want none", contractDiagnostics)
	}
	bodyResult, err := operationRegistry.Dispatch(call, "replace_body", map[string]any{
		"file": implementationRelative, "symbol": "InMemoryAuditSink.record(LedgerEntry)", "body": "return;",
		"language": "java", "trust_workspace": true, "jdtls_home": jdtlsHome, "java_bin": javaBin, "import_maven": true,
	})
	if err != nil {
		t.Fatalf("real semantic_replace_body Java: %v", err)
	}
	if _, ok := bodyResult.(operation.FileEditRes); !ok {
		t.Fatalf("real Java body replacement result has type %T", bodyResult)
	}
	implementationSource, err = os.ReadFile(implementationFile)
	if err != nil || !strings.Contains(string(implementationSource), "record(LedgerEntry entry)") || !strings.Contains(string(implementationSource), "return;") {
		t.Fatalf("real semantic_replace_body did not update record implementation: source=%s err=%v", implementationSource, err)
	}
	methodRename, err := operationRegistry.Dispatch(call, "rename", map[string]any{
		"file":   filepath.ToSlash(filepath.Join("ledger-domain", "src/main/java/com/acme/ledger/domain/AuditSink.java")),
		"symbol": "AuditSink.record(LedgerEntry)", "to": "append", "language": "java", "trust_workspace": true,
		"jdtls_home": jdtlsHome, "java_bin": javaBin, "import_maven": true,
	})
	if err != nil {
		t.Fatalf("real semantic_rename Java interface method: %v", err)
	}
	if result, ok := methodRename.(*backend.RenameResult); !ok || result == nil || result.Diagnostics.NetDelta != 0 {
		t.Fatalf("real Java interface method rename result = %#v", methodRename)
	}
	for file, expected := range map[string]string{
		filepath.Join(root, "ledger-domain", "src/main/java/com/acme/ledger/domain/AuditSink.java"): "append(LedgerEntry entry)",
		serviceFile:        "auditSink.append(options.entry())",
		implementationFile: "append(LedgerEntry entry)",
	} {
		updated, readErr := os.ReadFile(file)
		if readErr != nil || !bytes.Contains(updated, []byte(expected)) {
			t.Fatalf("real semantic_rename did not update %s: source=%s err=%v", file, updated, readErr)
		}
	}
	publicTypeFile := filepath.Join(root, "ledger-domain", "src", "main", "java", "com", "acme", "ledger", "domain", "EntryType.java")
	publicRename, err := operationRegistry.Dispatch(call, "rename", map[string]any{
		"file":   filepath.ToSlash(filepath.Join("ledger-domain", "src/main/java/com/acme/ledger/domain/EntryType.java")),
		"symbol": "EntryType", "to": "PostingKind", "language": "java", "trust_workspace": true,
		"jdtls_home": jdtlsHome, "java_bin": javaBin, "import_maven": true,
	})
	if err != nil {
		t.Fatalf("real semantic_rename Java public type: %v", err)
	}
	publicRenameResult, ok := publicRename.(*backend.RenameResult)
	if !ok || publicRenameResult == nil || publicRenameResult.TargetFile != filepath.Join(filepath.Dir(publicTypeFile), "PostingKind.java") {
		t.Fatalf("real Java public-type rename result = %#v", publicRename)
	}
	if _, statErr := os.Stat(publicTypeFile); !os.IsNotExist(statErr) {
		t.Fatalf("real public-type rename left the old source file: err=%v", statErr)
	}
	postingKindFile := filepath.Join(filepath.Dir(publicTypeFile), "PostingKind.java")
	postingKindSource, err := os.ReadFile(postingKindFile)
	if err != nil || !bytes.Contains(postingKindSource, []byte("enum PostingKind")) {
		t.Fatalf("real public-type rename did not create PostingKind.java: source=%s err=%v", postingKindSource, err)
	}
	for _, file := range []string{
		filepath.Join(root, "ledger-domain", "src/main/java/com/acme/ledger/domain/LedgerEntry.java"),
		filepath.Join(root, "ledger-service", "src/main/java/com/acme/ledger/service/LedgerCaller.java"),
		filepath.Join(root, "ledger-service", "src/test/java/com/acme/ledger/service/LedgerServiceTest.java"),
	} {
		updated, readErr := os.ReadFile(file)
		if readErr != nil || !bytes.Contains(updated, []byte("PostingKind")) {
			t.Fatalf("real public-type rename did not update %s: source=%s err=%v", file, updated, readErr)
		}
	}
	managedDependency, err := operationRegistry.Dispatch(call, "add_build_dependency", map[string]any{
		"file": "ledger-service", "dependency": "org.slf4j:slf4j-api:2.0.12", "language": "java", "trust_workspace": true,
	})
	if err != nil {
		t.Fatalf("real semantic_add_build_dependency for managed Maven coordinate: %v", err)
	}
	if _, ok := managedDependency.(operation.BuildDependencyRes); !ok {
		t.Fatalf("real managed Maven dependency result has type %T", managedDependency)
	}
	servicePOMData, err := os.ReadFile(servicePOM)
	if err != nil {
		t.Fatalf("read module POM after managed dependency operation: %v", err)
	}
	managedBlock := javaDependencyBlock(servicePOMData, "slf4j-api")
	if managedBlock == "" || strings.Contains(managedBlock, "<version>") {
		t.Fatalf("managed module dependency is missing or duplicated its inherited version: %s", servicePOMData)
	}
	unmanagedDependency, err := operationRegistry.Dispatch(call, "add_build_dependency", map[string]any{
		"file": "ledger-service", "dependency": "com.google.guava:guava:33.0.0-jre", "language": "java", "trust_workspace": true,
	})
	if err != nil {
		t.Fatalf("real semantic_add_build_dependency for unmanaged Maven coordinate: %v", err)
	}
	if _, ok := unmanagedDependency.(operation.BuildDependencyRes); !ok {
		t.Fatalf("real unmanaged Maven dependency result has type %T", unmanagedDependency)
	}
	rootPOMData, err := os.ReadFile(filepath.Join(root, "pom.xml"))
	if err != nil {
		t.Fatalf("read root POM after unmanaged dependency operation: %v", err)
	}
	servicePOMData, err = os.ReadFile(servicePOM)
	if err != nil {
		t.Fatalf("read module POM after unmanaged dependency operation: %v", err)
	}
	if !bytes.Contains(rootPOMData, []byte("<artifactId>guava</artifactId>")) || !bytes.Contains(rootPOMData, []byte("<version>33.0.0-jre</version>")) {
		t.Fatalf("unmanaged dependency was not added to root dependency management: %s", rootPOMData)
	}
	guavaBlock := javaDependencyBlock(servicePOMData, "guava")
	if guavaBlock == "" || strings.Contains(guavaBlock, "<version>") {
		t.Fatalf("managed child dependency is missing or duplicates its inherited version: %s", servicePOMData)
	}
	_, moveErr := operationRegistry.Dispatch(call, "move_construct", map[string]any{
		"file":             filepath.ToSlash(filepath.Join("ledger-domain", "src/main/java/com/acme/ledger/domain/AuditSink.java")),
		"symbol":           "AuditSink",
		"destination_file": filepath.ToSlash(filepath.Join("ledger-domain", "src/main/java/com/acme/ledger/domain/contracts/ContractTarget.java")),
		"target_symbol":    "ContractTarget",
		"language":         "java", "trust_workspace": true, "jdtls_home": jdtlsHome, "java_bin": javaBin, "import_maven": true,
	})
	if moveErr != nil {
		t.Fatalf("real semantic_move_construct Java public type: %v", moveErr)
	}
	oldTypeFile := filepath.Join(root, "ledger-domain", "src/main/java/com/acme/ledger/domain/AuditSink.java")
	newTypeFile := filepath.Join(root, "ledger-domain", "src/main/java/com/acme/ledger/domain/contracts/AuditSink.java")
	serviceAfterMove, readErr := os.ReadFile(serviceFile)
	if readErr != nil {
		t.Fatalf("read Java service after public type move: %v", readErr)
	}
	if _, statErr := os.Stat(oldTypeFile); !os.IsNotExist(statErr) {
		t.Fatalf("real Java type move left its old source file: %v", statErr)
	}
	movedSource, readErr := os.ReadFile(newTypeFile)
	if readErr != nil || !strings.Contains(string(movedSource), "package com.acme.ledger.domain.contracts;") || !strings.Contains(string(movedSource), "interface AuditSink") {
		t.Fatalf("real Java type move did not update the destination package: source=%s err=%v", movedSource, readErr)
	}
	if !strings.Contains(string(serviceAfterMove), "import com.acme.ledger.domain.contracts.AuditSink;") {
		t.Fatalf("real Java type move did not update the package caller: source=%s", serviceAfterMove)
	}
}

func javaDependencyBlock(data []byte, artifact string) string {
	artifactMarker := []byte("<artifactId>" + artifact + "</artifactId>")
	artifactIndex := bytes.Index(data, artifactMarker)
	if artifactIndex < 0 {
		return ""
	}
	start := bytes.LastIndex(data[:artifactIndex], []byte("<dependency>"))
	endOffset := bytes.Index(data[artifactIndex:], []byte("</dependency>"))
	if start < 0 || endOffset < 0 {
		return ""
	}
	end := artifactIndex + endOffset + len("</dependency>")
	return string(data[start:end])
}

func validateRealJavaRuntime(ctx context.Context, javaBin string) error {
	output, err := exec.CommandContext(ctx, javaBin, "-version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("run Java version check: %w: %s", err, output)
	}
	version := string(output)
	marker := `version "`
	index := strings.Index(version, marker)
	if index < 0 {
		return fmt.Errorf("java version output has no quoted version: %s", version)
	}
	version = version[index+len(marker):]
	majorText, _, _ := strings.Cut(version, ".")
	major, err := strconv.Atoi(majorText)
	if err != nil || major < 21 {
		return fmt.Errorf("java 21 or newer is required, got %q", majorText)
	}
	return nil
}

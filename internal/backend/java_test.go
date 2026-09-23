package backend_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"bytes"
	"semedit/internal/backend"
	javabackend "semedit/internal/backend/java"
)

type fakeJavaSession struct {
	symbols            json.RawMessage
	methods            []string
	requests           map[string]any
	initialize         map[string]any
	notifications      map[string]any
	cancel             bool
	closed             int
	closeErr           error
	formatting         json.RawMessage
	codeAction         json.RawMessage
	diagnostics        []backend.Diagnostic
	diagnosticsVersion int
	blockDiagnostics   bool
}

func (f *fakeJavaSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	f.methods = append(f.methods, method)
	if f.requests == nil {
		f.requests = make(map[string]any)
	}
	f.requests[method] = params
	if method == "initialize" {
		f.initialize = params.(map[string]any)
		return json.RawMessage(`{}`), nil
	}
	if method == "textDocument/formatting" && f.formatting != nil {
		return f.formatting, nil
	}
	if method == "textDocument/codeAction" && f.codeAction != nil {
		return f.codeAction, nil
	}
	if f.cancel {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.symbols, nil
}
func (f *fakeJavaSession) Notify(_ context.Context, method string, params any) error {
	f.methods = append(f.methods, method)
	if f.notifications == nil {
		f.notifications = make(map[string]any)
	}
	f.notifications[method] = params
	return nil
}
func (f *fakeJavaSession) WaitDiagnostics(ctx context.Context, _ string, version int) ([]backend.Diagnostic, error) {
	f.diagnosticsVersion = version
	if f.blockDiagnostics {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return f.diagnostics, nil
	}
}
func (f *fakeJavaSession) Close() error { f.closed++; return f.closeErr }

func javaFixture(t *testing.T, source string) (string, string) {
	t.Helper()
	root := t.TempDir()
	file := filepath.Join(root, "Thing.java")
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, file
}

func trustedJavaProject(root, file string) backend.ProjectContext {
	return backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
}

func TestJavaSessionFingerprintReusesAndRestarts(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte(`<project><modules><module>module</module></modules></project>`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "module"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "module", "pom.xml"), []byte(`<project/>`), 0o600); err != nil {
		t.Fatal(err)
	}
	created := make([]*fakeJavaSession, 0, 3)
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		session := &fakeJavaSession{symbols: json.RawMessage(`[{"name":"Thing","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":13}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":11}}}]`)}
		created = append(created, session)
		return session, nil
	})
	project := trustedJavaProject(root, file)
	if _, err := underTest.Lookup(context.Background(), project, "Thing"); err != nil {
		t.Fatal(err)
	}
	if _, err := underTest.Lookup(context.Background(), project, "Thing"); err != nil {
		t.Fatal(err)
	}
	if len(created) != 1 {
		t.Fatalf("unchanged descriptor created %d sessions", len(created))
	}
	if err := os.WriteFile(filepath.Join(root, "module", "pom.xml"), []byte(`<project><name>changed</name></project>`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := underTest.Lookup(context.Background(), project, "Thing"); err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 || created[0].closed != 1 {
		t.Fatalf("descriptor change sessions=%d closed=%d", len(created), created[0].closed)
	}
	project.Java.ImportMaven = true
	if _, err := underTest.Lookup(context.Background(), project, "Thing"); err != nil {
		t.Fatal(err)
	}
	if len(created) != 3 || created[1].closed != 1 {
		t.Fatalf("config change sessions=%d closed=%d", len(created), created[1].closed)
	}
	aliasProject := trustedJavaProject(root, file)
	aliasProject.Java = backend.JavaConfig{}
	aliasProject.JDTLSHome = "alias-jdtls"
	if _, err := underTest.Lookup(context.Background(), aliasProject, "Thing"); err != nil {
		t.Fatal(err)
	}
	if len(created) != 4 {
		t.Fatalf("alias config did not restart session: %d", len(created))
	}
}

func TestJavaSessionFingerprintCloseFailurePreventsReplacement(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte(`<project/>`), 0o600); err != nil {
		t.Fatal(err)
	}
	first := &fakeJavaSession{closeErr: errors.New("close failed"), symbols: json.RawMessage(`[]`)}
	created := 0
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		created++
		if created == 1 {
			return first, nil
		}
		return &fakeJavaSession{symbols: json.RawMessage(`[]`)}, nil
	})
	project := trustedJavaProject(root, file)
	if _, err := underTest.Lookup(context.Background(), project, "Thing"); err == nil {
		t.Fatal("expected lookup failure")
	}
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte(`<project><name>changed</name></project>`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := underTest.Lookup(context.Background(), project, "Thing")
	if err == nil || !strings.Contains(err.Error(), "close stale Java session") || created != 1 {
		t.Fatalf("err=%v created=%d", err, created)
	}
}

func TestJavaFingerprintRejectsModuleDescriptorOutsideRoot(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	outside := filepath.Join(root, "..", "outside-java-pom.xml")
	if err := os.WriteFile(outside, []byte(`<project/>`), 0o600); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(outside) }()
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte(`<project><modules><module>../outside-java-pom</module></modules></project>`), 0o600); err != nil {
		t.Fatal(err)
	}
	started := false
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		started = true
		return nil, nil
	})
	_, err := underTest.Lookup(context.Background(), trustedJavaProject(root, file), "Thing")
	if err == nil || !errors.Is(err, javabackend.ErrJavaFileOutsideWorkspace) || started {
		t.Fatalf("err=%v started=%v", err, started)
	}
}

func TestJavaLookupInitializesUTF16AndHierarchicalSymbols(t *testing.T) {
	root, file := javaFixture(t, "package p;\nclass 😀Thing {\n  int field;\n  void run() {}\n}\n")
	session := &fakeJavaSession{symbols: json.RawMessage(`[{"name":"p","kind":4,"range":{"start":{"line":0,"character":0},"end":{"line":4,"character":1}},"selectionRange":{"start":{"line":0,"character":8},"end":{"line":0,"character":9}},"children":[{"name":"😀Thing","kind":5,"range":{"start":{"line":1,"character":0},"end":{"line":4,"character":1}},"selectionRange":{"start":{"line":1,"character":6},"end":{"line":1,"character":14}},"children":[{"name":"field","kind":8,"range":{"start":{"line":2,"character":2},"end":{"line":2,"character":12}},"selectionRange":{"start":{"line":2,"character":6},"end":{"line":2,"character":11}}}]}]}]`)}
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	})
	result, err := underTest.Lookup(context.Background(), trustedJavaProject(root, file), "p.😀Thing.field")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if result.Kind != "field" || result.Offset == 0 || result.Location.Range.Start.Character != 6 || result.Location.Range.End.Character != 11 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(session.methods) < 4 || session.methods[0] != "initialize" || session.methods[1] != "initialized" || session.methods[2] != "workspace/didChangeConfiguration" || session.methods[3] != "textDocument/didOpen" {
		t.Fatalf("session lifecycle = %v", session.methods)
	}
	caps := session.initialize["capabilities"].(map[string]any)
	general := caps["general"].(map[string]any)
	if general["positionEncodings"].([]string)[0] != "utf-16" {
		t.Fatalf("missing UTF-16 capability: %#v", general)
	}
	documentSymbol := caps["textDocument"].(map[string]any)["documentSymbol"].(map[string]any)
	if documentSymbol["hierarchicalDocumentSymbolSupport"] != true {
		t.Fatalf("missing hierarchy capability: %#v", documentSymbol)
	}
	settings := session.initialize["settings"].(map[string]any)
	change := session.notifications["workspace/didChangeConfiguration"].(map[string]any)["settings"].(map[string]any)
	for name, values := range map[string]map[string]any{"initialize": settings, "configuration-change": change} {
		if values["java.import.maven.enabled"] != false || values["java.import.gradle.enabled"] != false || values["java.autobuild.enabled"] != false || values["java.import.generatesMetadataFilesAtProjectRoot"] != false {
			t.Fatalf("unsafe %s settings: %#v", name, values)
		}
	}
}

func TestJavaLookupRejectsTrustBeforeSessionFactory(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	started := false
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		started = true
		return nil, nil
	})
	_, err := underTest.Lookup(context.Background(), backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageJava}, "Thing")
	if !errors.Is(err, backend.ErrWorkspaceTrustRequired) || started {
		t.Fatalf("err=%v started=%v", err, started)
	}
}

func TestJavaServiceDiscoversNearestRootAndRejectsSameLevelAmbiguity(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "src", "Thing.java")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("class Thing {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	session := &fakeJavaSession{symbols: json.RawMessage(`[]`)}
	service, err := backend.NewRegistry(javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	result, err := backend.NewService(service).Lookup(context.Background(), backend.ProjectContext{File: file, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}, "Thing")
	if err == nil || result != nil || !errors.Is(err, javabackend.ErrJavaSymbolNotFound) {
		t.Fatalf("nearest root lookup = %#v, %v", result, err)
	}
	if err := os.WriteFile(filepath.Join(root, "build.gradle"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	}).Lookup(context.Background(), backend.ProjectContext{File: file, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}, "Thing")
	if !errors.Is(err, javabackend.ErrJavaWorkspaceAmbiguous) {
		t.Fatalf("ambiguity error = %v", err)
	}
}

func TestJavaMavenReactorDiscoveryAndExplicitImportSetting(t *testing.T) {
	root := t.TempDir()
	module := filepath.Join(root, "module")
	file := filepath.Join(module, "src", "Thing.java")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(root, "pom.xml"):   `<project><modelVersion>4.0.0</modelVersion><groupId>x</groupId><artifactId>root</artifactId><version>1</version><packaging>pom</packaging><modules><module>module</module></modules></project>`,
		filepath.Join(module, "pom.xml"): `<project><modelVersion>4.0.0</modelVersion><parent><groupId>x</groupId><artifactId>root</artifactId><version>1</version></parent><artifactId>module</artifactId></project>`,
		file:                             "class Thing {}\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	session := &fakeJavaSession{symbols: json.RawMessage(`[{"name":"Thing","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":13}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":11}}}]`)}
	var gotRoot string
	underTest := javabackend.NewJavaBackendWithFactory(func(_ context.Context, root string, config backend.JavaConfig) (javabackend.JavaSession, error) {
		gotRoot = root
		if !config.ImportMaven {
			t.Fatal("expected explicit Maven import opt-in")
		}
		return session, nil
	})
	project := backend.ProjectContext{File: file, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true), Java: backend.JavaConfig{ImportMaven: true}}
	if _, err := underTest.Lookup(context.Background(), project, "Thing"); err != nil {
		t.Fatal(err)
	}
	if gotRoot != backend.CanonicalWorkspaceRoot(root) {
		t.Fatalf("reactor root = %q, want %q", gotRoot, root)
	}
	if err := os.WriteFile(filepath.Join(root, "build.gradle"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := javabackend.NewJavaBackend().TrustedWorkspaceRoot(backend.ProjectContext{File: file}); !errors.Is(err, javabackend.ErrJavaWorkspaceAmbiguous) {
		t.Fatalf("reactor candidate ambiguity = %v", err)
	}
	settings := session.initialize["settings"].(map[string]any)
	if settings["java.import.maven.enabled"] != true || settings["java.import.gradle.enabled"] != false {
		t.Fatalf("unsafe import settings: %#v", settings)
	}
	if settings["java.import.generatesMetadataFilesAtProjectRoot"] != false {
		t.Fatalf("metadata files at project root must remain disabled: %#v", settings)
	}
	change := session.notifications["workspace/didChangeConfiguration"].(map[string]any)["settings"].(map[string]any)
	if change["java.import.maven.enabled"] != true || change["java.import.gradle.enabled"] != false || change["java.autobuild.enabled"] != false || change["java.import.generatesMetadataFilesAtProjectRoot"] != false {
		t.Fatalf("unsafe configuration-change settings: %#v", change)
	}
}

func TestJavaMavenInheritedParentDoesNotImplyReactorMembership(t *testing.T) {
	root := t.TempDir()
	module := filepath.Join(root, "module")
	file := filepath.Join(module, "src", "Thing.java")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(root, "pom.xml"):   `<project><packaging>pom</packaging></project>`,
		filepath.Join(module, "pom.xml"): `<project><parent><relativePath>../pom.xml</relativePath></parent></project>`,
		file:                             "class Thing {}\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := javabackend.NewJavaBackend().TrustedWorkspaceRoot(backend.ProjectContext{File: file})
	if err != nil {
		t.Fatal(err)
	}
	if got != backend.CanonicalWorkspaceRoot(module) {
		t.Fatalf("inherited non-reactor root = %q, want module %q", got, module)
	}
}

func TestJavaMavenReactorDiscoverySkipsNonPOMDirectories(t *testing.T) {
	root := t.TempDir()
	module := filepath.Join(root, "services", "widget")
	file := filepath.Join(module, "src", "Widget.java")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(root, "pom.xml"):   `<project><modules><module>services/widget</module></modules></project>`,
		filepath.Join(module, "pom.xml"): `<project></project>`,
		file:                             "class Widget {}\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := javabackend.NewJavaBackend().TrustedWorkspaceRoot(backend.ProjectContext{File: file})
	if err != nil {
		t.Fatal(err)
	}
	if got != backend.CanonicalWorkspaceRoot(root) {
		t.Fatalf("root = %q, want %q", got, root)
	}
}

func TestJavaMavenIgnoresUnrelatedPolyglotAncestor(t *testing.T) {
	root := t.TempDir()
	module := filepath.Join(root, "services", "widget")
	file := filepath.Join(module, "src", "Widget.java")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(root, "pom.xml"):      `<project><packaging>pom</packaging></project>`,
		filepath.Join(root, "build.gradle"): "",
		filepath.Join(module, "pom.xml"):    `<project></project>`,
		file:                                "class Widget {}\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := javabackend.NewJavaBackend().TrustedWorkspaceRoot(backend.ProjectContext{File: file})
	if err != nil {
		t.Fatal(err)
	}
	if got != backend.CanonicalWorkspaceRoot(module) {
		t.Fatalf("root = %q, want module %q", got, module)
	}
}

func TestJavaLookupRejectsMalformedAndOutOfRootResponses(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	for name, test := range map[string]struct {
		raw  json.RawMessage
		want error
	}{
		"flat":        {raw: json.RawMessage(`[{"name":"Thing","kind":5,"location":{}}]`), want: javabackend.ErrJavaUnsupportedResponse},
		"out-of-root": {raw: json.RawMessage(`[{"name":"Thing","kind":5,"uri":"file:///tmp/out.java","range":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}},"selectionRange":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}}}]`), want: javabackend.ErrJavaMalformedResponse},
	} {
		t.Run(name, func(t *testing.T) {
			underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
				return &fakeJavaSession{symbols: test.raw}, nil
			})
			_, err := underTest.Lookup(context.Background(), trustedJavaProject(root, file), "Thing")
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestJavaLookupPropagatesCancellation(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	session := &fakeJavaSession{cancel: true}
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := underTest.Lookup(ctx, trustedJavaProject(root, file), "Thing")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancellation error = %v", err)
	}
}

func TestJavaVerifyFormattingWritesAndForwardsDiagnostics(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	session := &fakeJavaSession{
		formatting:  json.RawMessage(`[{"range":{"start":{"line":0,"character":6},"end":{"line":0,"character":11}},"newText":"Gadget"}]`),
		diagnostics: []backend.Diagnostic{{Message: "warning", Severity: 2}},
	}
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	})
	result, err := underTest.Verify(context.Background(), backend.VerifyRequest{
		Project:            trustedJavaProject(root, file),
		FormatSelectedFile: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "class Gadget {}\n" {
		t.Fatalf("formatted source = %q", contents)
	}
	if len(result) != 1 || result[0].Message != "warning" {
		t.Fatalf("diagnostics = %#v", result)
	}
	for _, method := range []string{"initialize", "textDocument/didOpen", "textDocument/formatting", "textDocument/didChange", "textDocument/didSave"} {
		if !containsString(session.methods, method) {
			t.Fatalf("missing session method %q in %v", method, session.methods)
		}
	}
}

func TestJavaVerifyOrganizeImportsWritesSelectedFile(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	uri := (&url.URL{Scheme: "file", Path: file}).String()
	session := &fakeJavaSession{codeAction: json.RawMessage(fmt.Sprintf(`[{"kind":"source.organizeImports","edit":{"changes":{%q:[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":0}},"newText":"import x.Y;\n"}]}}}]`, uri))}
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	})
	if _, err := underTest.Verify(context.Background(), backend.VerifyRequest{Project: trustedJavaProject(root, file), OrganizeImports: true}); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "import x.Y;\nclass Thing {}\n" {
		t.Fatalf("organized source = %q", contents)
	}
	if !containsString(session.methods, "textDocument/codeAction") {
		t.Fatalf("codeAction was not requested: %v", session.methods)
	}
}

func TestJavaVerifyNoOpPreservesFileAndDoesNotNotifyMutation(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	before, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	uri := (&url.URL{Scheme: "file", Path: file}).String()
	session := &fakeJavaSession{
		formatting:  json.RawMessage(`[ {"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}},"newText":"class"} ]`),
		codeAction:  json.RawMessage(fmt.Sprintf(`[ {"kind":"source.organizeImports","edit":{"changes":{%q:[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}},"newText":"class"}]}}} ]`, uri)),
		diagnostics: []backend.Diagnostic{{Message: "warning", Severity: 2}},
	}
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	})
	result, err := underTest.Verify(context.Background(), backend.VerifyRequest{
		Project:            trustedJavaProject(root, file),
		FormatSelectedFile: true,
		OrganizeImports:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatalf("no-op formatting changed mtime from %v to %v", before.ModTime(), after.ModTime())
	}
	if containsString(session.methods, "textDocument/didChange") || containsString(session.methods, "textDocument/didSave") {
		t.Fatalf("no-op formatting fabricated mutation notification: %v", session.methods)
	}
	if session.diagnosticsVersion != 1 {
		t.Fatalf("diagnostics readiness version = %d, want 1", session.diagnosticsVersion)
	}
	if params, ok := session.requests["textDocument/codeAction"].(map[string]any); !ok || params["textDocument"].(map[string]any)["version"] != 1 {
		t.Fatalf("no-op codeAction did not target version 1: %#v", session.requests["textDocument/codeAction"])
	}
	if len(result) != 1 || result[0].Message != "warning" {
		t.Fatalf("diagnostics = %#v", result)
	}
}

func TestJavaVerifySurfacesCloseFailureAfterSuccess(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	session := &fakeJavaSession{
		formatting: json.RawMessage(`[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":0}},"newText":""}]`),
		closeErr:   errors.New("close failed"),
	}
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	})
	_, err := underTest.Verify(context.Background(), backend.VerifyRequest{Project: trustedJavaProject(root, file), FormatSelectedFile: true})
	if err == nil || !strings.Contains(err.Error(), "close Java session") {
		t.Fatalf("close error = %v", err)
	}
}

func TestJavaVerifyCombinedActionsSynchronizeFormattingVersion(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	uri := (&url.URL{Scheme: "file", Path: file}).String()
	session := &fakeJavaSession{
		formatting: json.RawMessage(`[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":0}},"newText":"// formatted\n"}]`),
		codeAction: json.RawMessage(fmt.Sprintf(`[{"kind":"source.organizeImports","edit":{"documentChanges":[{"textDocument":{"uri":%q,"version":2},"edits":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":0}},"newText":"import x.Y;\n"}]}]}}]`, uri)),
	}
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	})
	if _, err := underTest.Verify(context.Background(), backend.VerifyRequest{Project: trustedJavaProject(root, file), FormatSelectedFile: true, OrganizeImports: true}); err != nil {
		t.Fatal(err)
	}
	params, ok := session.requests["textDocument/codeAction"].(map[string]any)
	if !ok || params["textDocument"].(map[string]any)["version"] != 2 {
		t.Fatalf("codeAction did not target synchronized version 2: %#v", session.requests["textDocument/codeAction"])
	}
}

func TestJavaVerifyInvalidFormatterEditClosesSession(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	session := &fakeJavaSession{formatting: json.RawMessage(`[{"range":{"start":{"line":0,"character":99},"end":{"line":0,"character":100}},"newText":"x"}]`)}
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	})
	if _, err := underTest.Verify(context.Background(), backend.VerifyRequest{Project: trustedJavaProject(root, file), FormatSelectedFile: true}); err == nil {
		t.Fatal("invalid formatting edit unexpectedly succeeded")
	}
	if session.closed != 1 {
		t.Fatalf("invalid edit closed session %d times, want 1", session.closed)
	}
}

func TestJavaVerifyDiagnosticsTimeoutIsDistinct(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	session := &fakeJavaSession{
		formatting:       json.RawMessage(`[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":0}},"newText":""}]`),
		blockDiagnostics: true,
	}
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := underTest.Verify(ctx, backend.VerifyRequest{Project: trustedJavaProject(root, file), FormatSelectedFile: true})
	if !errors.Is(err, javabackend.ErrJavaDiagnosticsTimeout) {
		t.Fatalf("timeout error = %v", err)
	}
}

func TestJavaVerifyRejectsUnsafeRequestsWithoutWriting(t *testing.T) {
	tests := []struct {
		name       string
		project    func(root, file string) backend.ProjectContext
		formatting json.RawMessage
		codeAction json.RawMessage
		format     bool
		organize   bool
	}{
		{name: "no actions", project: trustedJavaProject},
		{name: "untrusted", project: func(root, file string) backend.ProjectContext {
			return backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageJava}
		}, format: true},
		{name: "non java", project: func(root, file string) backend.ProjectContext {
			p := trustedJavaProject(root, file)
			p.File = filepath.Join(root, "Thing.txt")
			return p
		}, format: true},
		{name: "overlap formatting", project: trustedJavaProject, formatting: json.RawMessage(`[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":6}},"newText":"x"},{"range":{"start":{"line":0,"character":5},"end":{"line":0,"character":11}},"newText":"y"}]`), format: true},
		{name: "organize command", project: trustedJavaProject, codeAction: json.RawMessage(`[{"kind":"source.organizeImports","command":{"title":"run"}}]`), organize: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, file := javaFixture(t, "class Thing {}\n")
			original, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			started := false
			session := &fakeJavaSession{formatting: tc.formatting, codeAction: tc.codeAction}
			underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
				started = true
				return session, nil
			})
			_, err = underTest.Verify(context.Background(), backend.VerifyRequest{Project: tc.project(root, file), FormatSelectedFile: tc.format, OrganizeImports: tc.organize})
			if err == nil {
				t.Fatal("Verify unexpectedly succeeded")
			}
			contents, readErr := os.ReadFile(file)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(contents, original) {
				t.Fatalf("unsafe request changed file to %q", contents)
			}
			if tc.name == "no actions" || tc.name == "untrusted" || tc.name == "non java" {
				if started {
					t.Fatal("unsafe request started a Java session")
				}
			}
		})
	}
}

func containsString(values []string, want string) bool {
	return slices.Contains(values, want)
}

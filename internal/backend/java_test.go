package backend_test

import (
	"context"
	"crypto/sha256"
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
	"semedit/internal/backend/pathutil"
	"semedit/internal/pipeline"
)

type fakeJavaSession struct {
	symbols              json.RawMessage
	methods              []string
	requests             map[string]any
	initialize           map[string]any
	notifications        map[string]any
	cancel               bool
	closed               int
	closeErr             error
	formatting           json.RawMessage
	codeAction           json.RawMessage
	prepareRename        json.RawMessage
	renameResult         json.RawMessage
	requestErrors        map[string]error
	notifyErrors         map[string]error
	diagnostics          []backend.Diagnostic
	diagnosticsErr       error
	diagnosticsVersion   int
	waitDiagnosticsCalls int
	blockDiagnostics     bool
}

func (f *fakeJavaSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	f.methods = append(f.methods, method)
	if f.requests == nil {
		f.requests = make(map[string]any)
	}
	f.requests[method] = params
	if err := f.requestErrors[method]; err != nil {
		return nil, err
	}
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
	if method == "textDocument/prepareRename" && f.prepareRename != nil {
		return f.prepareRename, nil
	}
	if method == "textDocument/rename" && f.renameResult != nil {
		return f.renameResult, nil
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
	return f.notifyErrors[method]
}
func (f *fakeJavaSession) WaitDiagnostics(ctx context.Context, _ string, version int) ([]backend.Diagnostic, error) {
	f.waitDiagnosticsCalls++
	f.diagnosticsVersion = version
	if f.diagnosticsErr != nil {
		return nil, f.diagnosticsErr
	}
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

//nolint:dupl // Keep the Java backend response contract explicit beside its fixture.
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
	if session.notifications["textDocument/didClose"] == nil {
		t.Fatal("canceled lookup did not close the opened document")
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

type statefulJavaSession struct {
	open        map[string]string
	events      []string
	closed      int
	diagnostics []backend.Diagnostic
	waitCalls   int
}

func (s *statefulJavaSession) Request(_ context.Context, method string, params any) (json.RawMessage, error) {
	if method == "initialize" {
		s.events = append(s.events, "request:initialize")
		return json.RawMessage(`{}`), nil
	}
	if method != "textDocument/documentSymbol" {
		s.events = append(s.events, "request:"+method)
		return json.RawMessage(`[]`), nil
	}
	document := params.(map[string]any)["textDocument"].(map[string]string)
	uri := document["uri"]
	source, ok := s.open[uri]
	if !ok {
		return nil, fmt.Errorf("documentSymbol requested without didOpen")
	}
	s.events = append(s.events, "request:"+source)
	line := strings.TrimSuffix(source, "\n")
	fields := strings.Fields(strings.TrimPrefix(line, "class "))
	if len(fields) == 0 {
		return nil, fmt.Errorf("missing class name in opened source")
	}
	name := fields[0]
	value := map[string]any{
		"name":           name,
		"kind":           5,
		"range":          map[string]any{"start": map[string]int{"line": 0, "character": 0}, "end": map[string]int{"line": 0, "character": len(line)}},
		"selectionRange": map[string]any{"start": map[string]int{"line": 0, "character": 6}, "end": map[string]int{"line": 0, "character": 6 + len(name)}},
	}
	raw, err := json.Marshal([]any{value})
	return raw, err
}

func (s *statefulJavaSession) Notify(_ context.Context, method string, params any) error {
	if method != "textDocument/didOpen" && method != "textDocument/didClose" {
		s.events = append(s.events, method)
		return nil
	}
	document := params.(map[string]any)["textDocument"]
	var uri string
	switch value := document.(type) {
	case map[string]any:
		uri = value["uri"].(string)
	case map[string]string:
		uri = value["uri"]
	default:
		return fmt.Errorf("unexpected document params %T", document)
	}
	switch method {
	case "textDocument/didOpen":
		if s.open == nil {
			s.open = make(map[string]string)
		}
		if _, exists := s.open[uri]; exists {
			return fmt.Errorf("duplicate didOpen for %s", uri)
		}
		source := document.(map[string]any)["text"].(string)
		s.open[uri] = source
		s.events = append(s.events, "open:"+source)
	case "textDocument/didClose":
		source, exists := s.open[uri]
		if !exists {
			return fmt.Errorf("didClose without didOpen for %s", uri)
		}
		delete(s.open, uri)
		s.events = append(s.events, "close:"+source)
	}
	return nil
}

func (s *statefulJavaSession) WaitDiagnostics(_ context.Context, _ string, _ int) ([]backend.Diagnostic, error) {
	s.waitCalls++
	return s.diagnostics, nil
}

func (s *statefulJavaSession) Close() error {
	s.closed++
	return nil
}

func TestJavaReadOperationsUseFreshDocumentSnapshotsAndWarmSession(t *testing.T) {
	root, file := javaFixture(t, "class Alpha {}\n")
	session := &statefulJavaSession{}
	factoryCalls := 0
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		factoryCalls++
		return session, nil
	})
	project := trustedJavaProject(root, file)
	if _, err := underTest.Lookup(context.Background(), project, "Alpha"); err != nil {
		t.Fatalf("lookup original snapshot: %v", err)
	}
	updated := []byte("class Bravo {}\n")
	if len(updated) != len([]byte("class Alpha {}\n")) {
		t.Fatal("test edit must preserve source length")
	}
	if err := pipeline.WriteAtomic(file, updated); err != nil {
		t.Fatal(err)
	}
	outline, err := underTest.Outline(context.Background(), backend.OutlineRequest{Project: project, Path: file, IncludeUnexported: true})
	if err != nil {
		t.Fatalf("outline edited snapshot: %v", err)
	}
	inspection, err := underTest.Inspect(context.Background(), backend.InspectRequest{Project: project, Symbol: "Bravo"})
	if err != nil {
		t.Fatalf("inspect edited snapshot: %v", err)
	}
	digest := sha256.Sum256(updated)
	wantRevision := fmt.Sprintf("%x", digest)
	if outline.Files[0].Revision != wantRevision || outline.Files[0].Symbols[0].Name != "Bravo" {
		t.Fatalf("outline reused stale source: %#v", outline.Files[0])
	}
	if len(inspection.Matches) != 1 || inspection.Matches[0].Name != "Bravo" || inspection.Matches[0].Source != "class Bravo {}" || inspection.Matches[0].Revision != wantRevision || inspection.Matches[0].SourceExtent != "declaration" {
		t.Fatalf("inspection did not use edited source: %#v", inspection)
	}
	wantEvents := []string{
		"open:class Alpha {}\n", "request:class Alpha {}\n", "close:class Alpha {}\n",
		"open:class Bravo {}\n", "request:class Bravo {}\n", "close:class Bravo {}\n",
		"open:class Bravo {}\n", "request:class Bravo {}\n", "close:class Bravo {}\n",
	}
	if len(session.events) < len(wantEvents) || !slices.Equal(session.events[len(session.events)-len(wantEvents):], wantEvents) {
		t.Fatalf("document lifecycle events = %#v, want suffix %#v", session.events, wantEvents)
	}
	if factoryCalls != 1 || len(session.open) != 0 {
		t.Fatalf("successful reads must retain one warm session and close documents: factories=%d open=%v", factoryCalls, session.open)
	}
	if err := underTest.Close(); err != nil || session.closed != 1 {
		t.Fatalf("final close err=%v closed=%d", err, session.closed)
	}
}

func TestJavaReadDocumentFailuresInvalidateSessionAndJoinCleanup(t *testing.T) {
	openErr := errors.New("didOpen failed")
	requestErr := errors.New("symbol request failed")
	documentCloseErr := errors.New("didClose failed")
	sessionCloseErr := errors.New("session close failed")
	validSymbols := json.RawMessage(`[{"name":"Thing","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":14}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":11}}}]`)
	tests := []struct {
		name       string
		notifyErrs map[string]error
		requestErr error
		wantErrors []error
	}{
		{name: "open and session close", notifyErrs: map[string]error{"textDocument/didOpen": openErr}, wantErrors: []error{openErr, sessionCloseErr}},
		{name: "close and session close", notifyErrs: map[string]error{"textDocument/didClose": documentCloseErr}, wantErrors: []error{documentCloseErr, sessionCloseErr}},
		{name: "request and close failures", notifyErrs: map[string]error{"textDocument/didClose": documentCloseErr}, requestErr: requestErr, wantErrors: []error{requestErr, documentCloseErr, sessionCloseErr}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, file := javaFixture(t, "class Thing {}\n")
			first := &fakeJavaSession{symbols: validSymbols, closeErr: sessionCloseErr, notifyErrors: test.notifyErrs}
			if test.requestErr != nil {
				first.requestErrors = map[string]error{"textDocument/documentSymbol": test.requestErr}
			}
			second := &fakeJavaSession{symbols: validSymbols}
			created := 0
			underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
				created++
				if created == 1 {
					return first, nil
				}
				return second, nil
			})
			project := trustedJavaProject(root, file)
			if _, err := underTest.Lookup(context.Background(), project, "Thing"); err == nil {
				t.Fatal("expected the first read to fail")
			} else {
				for _, want := range test.wantErrors {
					if !errors.Is(err, want) {
						t.Errorf("error %v does not include %v", err, want)
					}
				}
			}
			if first.closed != 1 {
				t.Fatalf("uncertain session was not retired: closes=%d", first.closed)
			}
			if _, err := underTest.Lookup(context.Background(), project, "Thing"); err != nil {
				t.Fatalf("read after retirement: %v", err)
			}
			if created != 2 {
				t.Fatalf("read after failure reused uncertain session: factories=%d", created)
			}
		})
	}
}

func TestJavaFailedRenameRetiresSessionBeforeNextRead(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	original, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	symbols := json.RawMessage(`[{"name":"Thing","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":14}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":11}}}]`)
	first := &fakeJavaSession{symbols: symbols, prepareRename: json.RawMessage(`null`)}
	second := &fakeJavaSession{symbols: symbols}
	created := 0
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		created++
		if created == 1 {
			return first, nil
		}
		return second, nil
	})
	project := trustedJavaProject(root, file)
	_, renameErr := underTest.Rename(context.Background(), backend.RenameRequest{Project: project, Symbol: "Thing", To: "Other"})
	if !errors.Is(renameErr, javabackend.ErrJavaRenameInvalidEdit) {
		t.Fatalf("failed rename error = %v", renameErr)
	}
	after, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(after, original) {
		t.Fatalf("failed rename changed source: %q err=%v", after, err)
	}
	if first.closed != 1 {
		t.Fatalf("failed rename did not retire session: %d", first.closed)
	}
	if _, err := underTest.Lookup(context.Background(), project, "Thing"); err != nil {
		t.Fatalf("read after failed rename: %v", err)
	}
	if created != 2 {
		t.Fatalf("read after failed rename reused session: %d", created)
	}
	if err := underTest.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestJavaVerifyUsesFreshSessionAfterReadForSameVersionDiagnostics(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	symbols := json.RawMessage(`[{"name":"Thing","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":14}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":11}}}]`)
	stale := &fakeJavaSession{symbols: symbols, diagnostics: []backend.Diagnostic{{Message: "stale v1"}}}
	fresh := &fakeJavaSession{symbols: symbols, formatting: json.RawMessage(`[]`), diagnostics: []backend.Diagnostic{{Message: "fresh v1"}}}
	created := 0
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		created++
		if created == 1 {
			return stale, nil
		}
		return fresh, nil
	})
	project := trustedJavaProject(root, file)
	if _, err := underTest.Lookup(context.Background(), project, "Thing"); err != nil {
		t.Fatal(err)
	}
	diagnostics, err := underTest.Verify(context.Background(), backend.VerifyRequest{Project: project, FormatSelectedFile: true})
	if err != nil {
		t.Fatalf("verify after lookup: %v", err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Message != "fresh v1" {
		t.Fatalf("verify accepted an old same-URI/version receipt: %#v", diagnostics)
	}
	if stale.waitDiagnosticsCalls != 0 || stale.closed != 1 || fresh.waitDiagnosticsCalls != 1 || fresh.closed != 1 || created != 2 {
		t.Fatalf("read/verify session lifecycle stale=%+v fresh=%+v created=%d", stale, fresh, created)
	}
}

func TestJavaOutlineInspectPreserveOverloadsAndServerDetail(t *testing.T) {
	source := "class Sample { void run(){} int run(int n){return n;} }\n"
	root, file := javaFixture(t, source)
	session := &fakeJavaSession{symbols: json.RawMessage(`[{"name":"Sample","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":55}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":12}},"children":[{"name":"run","kind":6,"detail":"method detail","range":{"start":{"line":0,"character":15},"end":{"line":0,"character":27}},"selectionRange":{"start":{"line":0,"character":20},"end":{"line":0,"character":23}}},{"name":"run","kind":6,"detail":"overload detail","range":{"start":{"line":0,"character":28},"end":{"line":0,"character":53}},"selectionRange":{"start":{"line":0,"character":32},"end":{"line":0,"character":35}}}]}]`)}
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	})
	project := trustedJavaProject(root, file)
	outline, err := underTest.Outline(context.Background(), backend.OutlineRequest{Project: project, Path: file, Kinds: []string{"method"}, IncludeUnexported: true})
	if err != nil {
		t.Fatalf("outline: %v", err)
	}
	if len(outline.Files) != 1 || len(outline.Files[0].Symbols) != 1 || outline.Files[0].Symbols[0].Kind != "class" || len(outline.Files[0].Symbols[0].Children) != 2 {
		t.Fatalf("method filtering did not retain class context and overloads: %#v", outline)
	}
	if outline.Files[0].Package != "" || outline.Files[0].Doc != nil || outline.Files[0].Imports != nil {
		t.Fatalf("Java invented unavailable metadata: %#v", outline.Files[0])
	}
	first := outline.Files[0].Symbols[0].Children[0]
	if first.Kind != "method" || first.Signature != nil || first.ServerDetail == nil || *first.ServerDetail != "method detail" {
		t.Fatalf("JDT LS detail was misrepresented: %#v", first)
	}
	inspection, err := underTest.Inspect(context.Background(), backend.InspectRequest{Project: project, Symbol: "Sample.run"})
	if err != nil {
		t.Fatalf("inspect overloads: %v", err)
	}
	if len(inspection.Matches) != 2 || inspection.Matches[0].Source != "void run(){}" || inspection.Matches[1].Source != "int run(int n){return n;}" {
		t.Fatalf("inspect did not preserve exact overload declaration spans: %#v", inspection.Matches)
	}
	if inspection.Matches[0].SourceExtent != "declaration" || inspection.Matches[0].File != "Thing.java" {
		t.Fatalf("unexpected inspect metadata: %#v", inspection.Matches[0])
	}
	if _, err := underTest.Inspect(context.Background(), backend.InspectRequest{Project: project, Symbol: "Missing"}); !errors.Is(err, javabackend.ErrJavaSymbolNotFound) {
		t.Fatalf("missing inspect target error = %v", err)
	}
}

func TestJavaOutlineRejectsDirectoryAndUnsupportedVisibilityBeforeLaunch(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	started := false
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		started = true
		return &fakeJavaSession{}, nil
	})
	project := trustedJavaProject(root, file)
	_, directoryErr := underTest.Outline(context.Background(), backend.OutlineRequest{Project: project, Path: root, IncludeUnexported: true})
	if directoryErr == nil || !strings.Contains(directoryErr.Error(), "selected files only") || started {
		t.Fatalf("directory outline err=%v started=%v", directoryErr, started)
	}
	_, visibilityErr := underTest.Outline(context.Background(), backend.OutlineRequest{Project: project, Path: file})
	if visibilityErr == nil || !strings.Contains(visibilityErr.Error(), "visibility filtering") || started {
		t.Fatalf("visibility-filtered outline err=%v started=%v", visibilityErr, started)
	}
}

func TestJavaReadRejectsSiblingDocumentURI(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	sibling := filepath.Join(root, "Sibling.java")
	session := &fakeJavaSession{symbols: json.RawMessage(`[{"name":"Thing","kind":5,"uri":"` + pathutil.FileURI(sibling) + `","range":{"start":{"line":0,"character":0},"end":{"line":0,"character":14}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":11}}}]`)}
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	})
	project := trustedJavaProject(root, file)
	_, err := underTest.Inspect(context.Background(), backend.InspectRequest{Project: project, Symbol: "Thing"})
	if !errors.Is(err, javabackend.ErrJavaMalformedResponse) {
		t.Fatalf("sibling URI was accepted: %v", err)
	}
	if session.notifications["textDocument/didClose"] == nil {
		t.Fatal("malformed response did not close the opened document")
	}
}

func TestJavaReadHandlesCROnlySourceRanges(t *testing.T) {
	source := "class Cr {}\rclass Next {}\r"
	root, file := javaFixture(t, source)
	session := &fakeJavaSession{symbols: json.RawMessage(`[{"name":"Cr","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":11}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":8}}},{"name":"Next","kind":5,"range":{"start":{"line":1,"character":0},"end":{"line":1,"character":13}},"selectionRange":{"start":{"line":1,"character":6},"end":{"line":1,"character":10}}}]`)}
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	})
	result, err := underTest.Inspect(context.Background(), backend.InspectRequest{Project: trustedJavaProject(root, file), Symbol: "Next"})
	if err != nil {
		t.Fatalf("CR-only source range: %v", err)
	}
	if len(result.Matches) != 1 || result.Matches[0].Source != "class Next {}" {
		t.Fatalf("incorrect CR-only declaration slice: %#v", result.Matches)
	}
}

func TestJavaVerifyJoinsDiagnosticsAndCloseFailures(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	diagnosticErr := errors.New("diagnostics failed")
	closeErr := errors.New("session close failed")
	session := &fakeJavaSession{
		formatting:     json.RawMessage(`[]`),
		diagnosticsErr: diagnosticErr,
		closeErr:       closeErr,
	}
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		return session, nil
	})
	_, err := underTest.Verify(context.Background(), backend.VerifyRequest{Project: trustedJavaProject(root, file), FormatSelectedFile: true})
	if !errors.Is(err, diagnosticErr) || !errors.Is(err, closeErr) || session.waitDiagnosticsCalls != 1 {
		t.Fatalf("verify lost diagnostics or close error: err=%v waits=%d", err, session.waitDiagnosticsCalls)
	}
	if session.closed != 1 {
		t.Fatalf("failed verify closed session %d times", session.closed)
	}
}

func TestJavaReadOperationsRejectUntrustedRequestsBeforeStarting(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	started := false
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		started = true
		return &fakeJavaSession{}, nil
	})
	untrusted := backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageJava}
	_, outlineErr := underTest.Outline(context.Background(), backend.OutlineRequest{Project: untrusted, Path: file, IncludeUnexported: true})
	_, inspectErr := underTest.Inspect(context.Background(), backend.InspectRequest{Project: untrusted, Symbol: "Thing"})
	if !errors.Is(outlineErr, backend.ErrWorkspaceTrustRequired) || !errors.Is(inspectErr, backend.ErrWorkspaceTrustRequired) || started {
		t.Fatalf("untrusted reads err=(%v,%v) started=%v", outlineErr, inspectErr, started)
	}
}

func TestJavaSuccessfulRenameClosesSessionBeforeReadUsesUpdatedSource(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	oldSymbols := json.RawMessage(`[{"name":"Thing","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":14}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":11}}}]`)
	newSymbols := json.RawMessage(`[{"name":"Gadget","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":15}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":12}}}]`)
	edit := map[string]any{"changes": map[string]any{pathutil.FileURI(file): []any{
		map[string]any{"range": map[string]any{"start": map[string]int{"line": 0, "character": 6}, "end": map[string]int{"line": 0, "character": 11}}, "newText": "Gadget"},
	}}}
	renameRaw, err := json.Marshal(edit)
	if err != nil {
		t.Fatal(err)
	}
	first := &fakeJavaSession{symbols: oldSymbols, prepareRename: json.RawMessage(`{"start":{"line":0,"character":6},"end":{"line":0,"character":11}}`), renameResult: renameRaw}
	second := &fakeJavaSession{symbols: newSymbols}
	created := 0
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		created++
		if created == 1 {
			return first, nil
		}
		return second, nil
	})
	project := trustedJavaProject(root, file)
	if _, err := underTest.Rename(context.Background(), backend.RenameRequest{Project: project, Symbol: "Thing", To: "Gadget"}); err != nil {
		t.Fatalf("rename: %v", err)
	}
	result, err := underTest.Inspect(context.Background(), backend.InspectRequest{Project: project, Symbol: "Gadget"})
	if err != nil {
		t.Fatalf("inspect after rename: %v", err)
	}
	if len(result.Matches) != 1 || result.Matches[0].Source != "class Gadget {}" || first.closed != 1 || created != 2 {
		t.Fatalf("renamed source or session transition incorrect: matches=%#v closed=%d factories=%d", result.Matches, first.closed, created)
	}
	if err := underTest.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestJavaReadSessionRejectsSecondWorkspace(t *testing.T) {
	rootA, fileA := javaFixture(t, "class Thing {}\n")
	rootB, fileB := javaFixture(t, "class Thing {}\n")
	symbols := json.RawMessage(`[{"name":"Thing","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":14}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":11}}}]`)
	created := 0
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		created++
		return &fakeJavaSession{symbols: symbols}, nil
	})
	if _, err := underTest.Lookup(context.Background(), trustedJavaProject(rootA, fileA), "Thing"); err != nil {
		t.Fatal(err)
	}
	_, err := underTest.Inspect(context.Background(), backend.InspectRequest{Project: trustedJavaProject(rootB, fileB), Symbol: "Thing"})
	if !errors.Is(err, javabackend.ErrJavaSessionConflict) || created != 1 {
		t.Fatalf("second workspace request err=%v factories=%d", err, created)
	}
	if err := underTest.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestJavaRenameCloseFailureKeepsCommittedResultAndResetsSession(t *testing.T) {
	root, file := javaFixture(t, "class Thing {}\n")
	oldSymbols := json.RawMessage(`[{"name":"Thing","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":14}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":11}}}]`)
	newSymbols := json.RawMessage(`[{"name":"Other","kind":5,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":14}},"selectionRange":{"start":{"line":0,"character":6},"end":{"line":0,"character":11}}}]`)
	edit := map[string]any{"changes": map[string]any{pathutil.FileURI(file): []any{
		map[string]any{"range": map[string]any{"start": map[string]int{"line": 0, "character": 6}, "end": map[string]int{"line": 0, "character": 11}}, "newText": "Other"},
	}}}
	renameRaw, err := json.Marshal(edit)
	if err != nil {
		t.Fatal(err)
	}
	closeErr := errors.New("rename session close failed")
	first := &fakeJavaSession{
		symbols:       oldSymbols,
		prepareRename: json.RawMessage(`{"start":{"line":0,"character":6},"end":{"line":0,"character":11}}`),
		renameResult:  renameRaw,
		closeErr:      closeErr,
	}
	second := &fakeJavaSession{symbols: newSymbols}
	created := 0
	underTest := javabackend.NewJavaBackendWithFactory(func(context.Context, string, backend.JavaConfig) (javabackend.JavaSession, error) {
		created++
		if created == 1 {
			return first, nil
		}
		return second, nil
	})
	project := trustedJavaProject(root, file)

	result, err := underTest.Rename(context.Background(), backend.RenameRequest{Project: project, Symbol: "Thing", To: "Other"})
	if err != nil || result == nil || result.Lookup == nil {
		t.Fatalf("Rename result = %+v, error = %v", result, err)
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "class Other {}\n" {
		t.Fatalf("committed source = %q", got)
	}
	if first.closed != 1 {
		t.Fatalf("original session closed %d times, want 1", first.closed)
	}

	if _, err := underTest.Lookup(context.Background(), project, "Other"); err != nil {
		t.Fatalf("Lookup after committed rename failed: %v", err)
	}
	if created != 2 || second.closed != 0 {
		t.Fatalf("sessions created=%d fresh session closed=%d, want 2 and 0", created, second.closed)
	}
	if err := underTest.Close(); err != nil {
		t.Fatal(err)
	}
}

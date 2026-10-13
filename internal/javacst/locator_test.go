// WHY: Exercises Java 21 construct discovery and exact byte ranges before the locator is wired into edits.
package javacst

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"semedit/internal/backend"
)

const javaMethod = `void run(int limit, int[] values) {
	outer: for (int i = 0; i < limit; i++) {
		if (i == 2) { hit(i); } else { miss(); }
	}
	for (int value : values) { use(value); }
	while (ready()) { tick(); }
	do { tick(); } while (ready());
	switch (limit) {
		case 1: hit(1); break;
		default: miss();
	}
	int result = switch (limit) { case 1 -> 1; default -> 0; };
	try (var resource = open()) { use(resource); }
	catch (IllegalStateException | IllegalArgumentException error) { fail(error); }
	synchronized (this) { tick(); }
	int[] copy = {1, 2, 3};
}`

func TestCandidatesEnumerateJava21Constructs(t *testing.T) {
	cases := []struct {
		kind backend.ConstructKind
		want int
	}{
		{backend.ConstructLoop, 4},
		{backend.ConstructIf, 1},
		{backend.ConstructElse, 1},
		{backend.ConstructCase, 4},
		{ConstructSwitch, 2},
		{backend.ConstructTryWithResources, 1},
		{backend.ConstructTryCatch, 1},
		{backend.ConstructSynchronized, 1},
		{ConstructElement, 3},
	}
	for _, test := range cases {
		t.Run(string(test.kind), func(t *testing.T) {
			got, err := Candidates([]byte(javaMethod), test.kind, "")
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != test.want {
				t.Fatalf("found %d candidates, want %d: %#v", len(got), test.want, got)
			}
			for _, candidate := range got {
				if candidate.StartByte >= candidate.EndByte || candidate.EndByte > uint32(len(javaMethod)) {
					t.Fatalf("invalid range: %#v", candidate)
				}
				if !strings.Contains(javaMethod[candidate.StartByte:candidate.EndByte], candidate.Syntax) {
					t.Fatalf("syntax %q not within range %#v", candidate.Syntax, candidate)
				}
			}
		})
	}
}

func TestElseAndArrayElementCandidatesPointAtExactSubranges(t *testing.T) {
	items, err := Candidates([]byte(`void run() { if (ready()) yes(); else no(); int[] a = {1, 2}; }`), backend.ConstructElse, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !strings.HasPrefix(items[0].Syntax, "else") {
		t.Fatalf("else candidate = %#v", items)
	}
	items, err = Candidates([]byte(`void run() { int[] a = {1, 2}; }`), ConstructElement, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Syntax != "1" || items[1].Syntax != "2" {
		t.Fatalf("element candidates = %#v", items)
	}
}

func TestCollectionFactoryElementCandidatesAndInsertion(t *testing.T) {
	source := []byte(`private final Map<String, Integer> limits = Map.of("DAILY", 1000);`)
	candidates, err := Candidates([]byte(`void run() { audit(Map.of("DAILY", 1000)); }`), ConstructElement, `Map.of("DAILY", 1000)`)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Syntax != `Map.of("DAILY", 1000)` {
		t.Fatalf("Map.of candidates = %#v", candidates)
	}
	updated, err := AppendJavaElement(source, `Map.of("DAILY", 1000)`, []byte(`"MONTHLY", 5000`))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(updated); !strings.Contains(got, `Map.of("DAILY", 1000, "MONTHLY", 5000)`) {
		t.Fatalf("Map.of insertion = %s", got)
	}

	for _, test := range []struct {
		name, source, discriminator, addition, want string
	}{
		{name: "Map.ofEntries", source: `private final Map<String, Integer> limits = Map.ofEntries(Map.entry("DAILY", 1000));`, discriminator: `Map.ofEntries(Map.entry("DAILY", 1000))`, addition: `Map.entry("MONTHLY", 5000)`, want: `Map.ofEntries(Map.entry("DAILY", 1000), Map.entry("MONTHLY", 5000))`},
		{name: "List.of", source: `private final List<String> names = List.of("daily");`, discriminator: `List.of("daily")`, addition: `"monthly"`, want: `List.of("daily", "monthly")`},
		{name: "Set.of", source: `private final Set<String> names = Set.of("daily");`, discriminator: `Set.of("daily")`, addition: `"monthly"`, want: `Set.of("daily", "monthly")`},
		{name: "empty factory", source: `private final List<String> names = List.of();`, discriminator: `List.of()`, addition: `"daily"`, want: `List.of("daily")`},
		{name: "array initializer", source: `private final int[] values = {1, 2};`, discriminator: `{1, 2}`, addition: `3`, want: `{1, 2, 3}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			updated, err := AppendJavaElement([]byte(test.source), test.discriminator, []byte(test.addition))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(updated), test.want) {
				t.Fatalf("insertion = %s, want %q", updated, test.want)
			}
		})
	}
}

func TestAppendJavaElementRefusesAmbiguousAndInvalidInsertions(t *testing.T) {
	declaration := []byte(`private final Map<String, Map<String, Integer>> limits = Map.of("DAILY", Map.of("WEEKLY", 4000));`)
	if _, err := AppendJavaElement(declaration, `Map.of`, []byte(`"MONTHLY", 5000`)); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("ambiguous initializer error = %v", err)
	}
	if _, err := AppendJavaElement(declaration, `Map.of("DAILY", Map.of("WEEKLY", 4000))`, []byte(`"MONTHLY"`)); err == nil {
		t.Fatal("Map.of accepted a key without a value")
	}
	if _, err := AppendJavaElement(declaration, `Map.of("MISSING")`, []byte(`"MONTHLY", 5000`)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unmatched initializer error = %v", err)
	}
}

func TestResolveConstructPathAndDiscriminator(t *testing.T) {
	source := []byte(`void run() { for (;;) { if (first()) one(); if (second()) two(); } }`)
	candidates, err := Candidates(source, backend.ConstructIf, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 || candidates[0].ConstructPath != "loop[0]/if[0]" || candidates[1].ConstructPath != "loop[0]/if[1]" {
		t.Fatalf("if candidates = %#v", candidates)
	}
	want, all, err := Resolve(source, backend.ConstructIf, "second()", "loop[0]/if[1]")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || want.StartByte != candidates[1].StartByte || !strings.Contains(want.Syntax, "second()") {
		t.Fatalf("resolved candidate = %#v, all = %#v", want, all)
	}
	if _, _, err := Resolve(source, backend.ConstructIf, "", "loop[0]/if[7]"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("invalid path error = %v", err)
	}
}

func TestResolveExpressionSelectsExactExpressionInsideStatement(t *testing.T) {
	method := []byte("void send(int id) { audit(id, true); }")
	candidate, _, err := ResolveExpression(method, "true", "")
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Syntax != "true" {
		t.Fatalf("selected syntax = %q, want the exact expression", candidate.Syntax)
	}
	updated := append(append([]byte(nil), method[:candidate.StartByte]...), []byte("enabled")...)
	updated = append(updated, method[candidate.EndByte:]...)
	if string(updated) != "void send(int id) { audit(id, enabled); }" {
		t.Fatalf("updated method = %s", updated)
	}
}

func TestCandidatesFailClosedOnMalformedMethod(t *testing.T) {
	if _, err := Candidates([]byte(`void run() { if (true) {`), backend.ConstructIf, ""); !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("malformed source error = %v", err)
	}
}

func TestStatementExpressionRangesCoverLambdaAndTextBlockSyntax(t *testing.T) {
	method := []byte("void run() {\n\tjava.util.function.Function<String, String> mapper = value -> value.trim();\n\tString message = \"\"\"\n\t\tready\n\t\t\"\"\";\n}")
	for _, discriminator := range []string{"value -> value.trim()", `"""`} {
		candidate, _, err := Resolve(method, backend.ConstructStatementOrExpr, discriminator, "")
		if err != nil {
			t.Fatalf("resolve %q: %v", discriminator, err)
		}
		start, offsetErr := ByteOffsetFromPosition(method, candidate.Range.Start)
		if offsetErr != nil || start != int(candidate.StartByte) {
			t.Fatalf("candidate range for %q starts at %d, expected byte %d: %v", discriminator, start, candidate.StartByte, offsetErr)
		}
		if !strings.Contains(candidate.Syntax, discriminator) {
			t.Fatalf("candidate for %q = %q", discriminator, candidate.Syntax)
		}
	}
}

func TestStatementExpressionCandidatesIncludeMethodInvocations(t *testing.T) {
	method := []byte("void run(String value) {\n\tString trimmed = value.trim();\n}")
	candidate, _, err := Resolve(method, backend.ConstructStatementOrExpr, "value.trim()", "")
	if err != nil {
		t.Fatalf("resolve method invocation: %v", err)
	}
	if candidate.Syntax != "value.trim()" {
		t.Fatalf("method invocation candidate = %q", candidate.Syntax)
	}
}

func TestRenameJavaIdentifiersInRangePreservesInterveningText(t *testing.T) {
	source := []byte("class Sample { void run() { int value = 1; // value stays in this comment\nString text = \"value stays in this string\"; use(value); } }")
	start := bytes.Index(source, []byte("int value"))
	end := bytes.Index(source, []byte(" } }"))
	if start < 0 || end < 0 {
		t.Fatal("could not locate rename range in fixture")
	}
	updated, err := RenameJavaIdentifiersInRange(source, start, end, "value", "amount")
	if err != nil {
		t.Fatalf("rename identifiers in range: %v", err)
	}
	if strings.Contains(string(updated), "int value") || !strings.Contains(string(updated), "int amount = 1; // value stays") || !strings.Contains(string(updated), "use(amount)") {
		t.Fatalf("renamed range = %q", updated)
	}
}

func TestCandidatesRequireStatementDiscriminator(t *testing.T) {
	if _, err := Candidates([]byte(`void run() { call(); }`), backend.ConstructStatementOrExpr, " "); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing discriminator error = %v", err)
	}
}

func TestCandidateRangeUsesUTF16Columns(t *testing.T) {
	source := []byte("void run() { String face = \"🙂\"; if (ready()) done(); }")
	items, err := Candidates(source, backend.ConstructIf, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("candidates = %#v", items)
	}
	wantStart := backend.PositionFromByteOffset(source, int(items[0].StartByte))
	wantEnd := backend.PositionFromByteOffset(source, int(items[0].EndByte))
	if items[0].Range.Start != wantStart || items[0].Range.End != wantEnd || int(items[0].StartByte)-items[0].Range.Start.Character != 2 {
		t.Fatalf("range = %#v, expected UTF-16 %v-%v", items[0].Range, wantStart, wantEnd)
	}
}

func TestSelectionRevisionTracksOwnerAndOrderedCandidates(t *testing.T) {
	items, err := Candidates([]byte(`void run() { if (ready()) done(); }`), backend.ConstructIf, "ready()")
	if err != nil {
		t.Fatal(err)
	}
	first, err := SelectionRevision("Widget.run()", backend.ConstructIf, "ready()", items)
	if err != nil {
		t.Fatal(err)
	}
	second, err := SelectionRevision("Widget.run()", backend.ConstructIf, "ready()", items)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("stable selection revision changed: %s != %s", first, second)
	}
	otherOwner, err := SelectionRevision("Other.run()", backend.ConstructIf, "ready()", items)
	if err != nil {
		t.Fatal(err)
	}
	if first == otherOwner {
		t.Fatal("selection revision did not bind the owner")
	}
}

func TestByteOffsetFromPositionUsesUTF16Units(t *testing.T) {
	source := []byte("🙂if (ready()) done();\nnext();")
	got, err := ByteOffsetFromPosition(source, backend.Position{Line: 0, Character: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got != len("🙂") {
		t.Fatalf("byte offset = %d, want %d", got, len("🙂"))
	}
	if _, err := ByteOffsetFromPosition(source, backend.Position{Line: 0, Character: 1}); err == nil {
		t.Fatal("position inside surrogate pair accepted")
	}
}

func TestReplaceCandidateParsesCompleteUpdatedMethod(t *testing.T) {
	method := []byte(`void run() { if (ready()) done(); }`)
	candidate, _, err := Resolve(method, backend.ConstructIf, "ready()", "")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := ReplaceCandidate(method, candidate, []byte(`if (ready()) { done(); next(); }`))
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) != `void run() { if (ready()) { done(); next(); } }` {
		t.Fatalf("updated method = %s", updated)
	}
	if _, err := ReplaceCandidate(method, candidate, []byte(`if (ready()) {`)); !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("invalid replacement error = %v", err)
	}
}

func TestInsertIntoMethodAtAnchors(t *testing.T) {
	method := []byte("void run() { work(); }")
	updated, err := InsertIntoMethod(method, "start", "", "", []byte("first();"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(updated), "first();") > strings.Index(string(updated), "work();") {
		t.Fatalf("start insertion order = %s", updated)
	}
	updated, err = InsertIntoMethod(method, "before", "work();", "", []byte("beforeWork();"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(updated), "beforeWork();") > strings.Index(string(updated), "work();") {
		t.Fatalf("before insertion order = %s", updated)
	}
	updated, err = InsertIntoMethod(method, "after", "work();", "", []byte("afterWork();"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Index(updated, []byte("afterWork();")) < bytes.Index(updated, []byte("work();")) {
		t.Fatalf("after insertion order = %s", updated)
	}
	method = []byte("void run() {\n  audit();\n  if (ready()) { work(); }\n}")
	updated, err = InsertIntoMethod(method, "after", "if (ready()) { work(); }", "", []byte("for (int i = 0; i < 1; i++) { audit(); }"))
	if err != nil {
		t.Fatalf("insert loop after conditional: %v", err)
	}
	if !bytes.Contains(updated, []byte("for (int i = 0; i < 1; i++) { audit(); }")) {
		t.Fatalf("inserted loop missing from method = %s", updated)
	}
	if validationErr := ValidateMethodConstructSnippet(backend.ConstructLoop, []byte("for (int i = 0; i < 1; i++) { audit(); }")); validationErr != nil {
		t.Fatalf("validate inserted loop: %v", validationErr)
	}
	method = []byte("void run() {\n    audit();\n    if (ready()) { work(); }\n  }")
	updated, err = InsertIntoMethod(method, "after", "if (ready()) { work(); }", "", []byte("for (int i = 0; i < 1; i++) { audit(); }"))
	if err != nil {
		t.Fatalf("insert after existing statement: %v", err)
	}
	if err := ValidateSource(updated); err != nil {
		t.Fatalf("updated method is invalid: %v\n%s", err, updated)
	}
	if _, err := InsertIntoMethod(method, "start", "", "", []byte("not valid Java !")); err == nil {
		t.Fatal("invalid Java insertion unexpectedly succeeded")
	}
}

func TestValidateMethodConstructSnippet(t *testing.T) {
	tests := []struct {
		name    string
		kind    backend.ConstructKind
		source  string
		wantErr bool
	}{
		{name: "statement", kind: backend.ConstructStatementOrExpr, source: "audit();"},
		{name: "loop", kind: backend.ConstructLoop, source: "for (int i = 0; i < 1; i++) { audit(); }"},
		{name: "conditional", kind: backend.ConstructIf, source: "if (ready) { audit(); }"},
		{name: "switch", kind: ConstructSwitch, source: "switch (state) { case 1 -> audit(); default -> skip(); }"},
		{name: "try with resources", kind: backend.ConstructTryWithResources, source: "try (var input = open()) { audit(); }"},
		{name: "synchronized", kind: backend.ConstructSynchronized, source: "synchronized (lock) { audit(); }"},
		{name: "wrong kind", kind: backend.ConstructIf, source: "while (ready) { audit(); }", wantErr: true},
		{name: "invalid syntax", kind: backend.ConstructLoop, source: "for ( {", wantErr: true},
		{name: "unsupported kind", kind: backend.ConstructCase, source: "case 1 -> audit();", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMethodConstructSnippet(tt.kind, []byte(tt.source))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateMethodConstructSnippet(%s, %q) error = %v, wantErr %t", tt.kind, tt.source, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDeclarationReplacement(t *testing.T) {
	tests := []struct {
		name        string
		existing    string
		replacement string
		wantErr     bool
	}{
		{name: "method body", existing: "void run() { oldWork(); }", replacement: "void run() { newWork(); }"},
		{name: "field initializer", existing: "private int limit = 10;", replacement: "private int limit = 20;"},
		{name: "multi field preserves names", existing: "private int low, high;", replacement: "private int low = 1, high = 2;"},
		{name: "renamed symbol", existing: "void run() {}", replacement: "void execute() {}", wantErr: true},
		{name: "changed declaration kind", existing: "private int limit;", replacement: "void limit() {}", wantErr: true},
		{name: "invalid syntax", existing: "private int limit;", replacement: "private int = ;", wantErr: true},
		{name: "multiple declarations", existing: "private int limit;", replacement: "private int limit; private int other;", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeclarationReplacement([]byte(tt.existing), []byte(tt.replacement))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateDeclarationReplacement error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

func TestRemoveCandidate(t *testing.T) {
	method := []byte("void run() {\n  if (ready()) { work(); }\n  audit();\n}")
	candidate, _, err := Resolve(method, backend.ConstructIf, "ready", "")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := RemoveCandidate(method, candidate)
	if err != nil {
		t.Fatal(err)
	}
	want := "void run() {\n  audit();\n}"
	if string(updated) != want {
		t.Fatalf("removed method = %q, want %q", updated, want)
	}
}

func TestRemoveJavaDeclarationIncludesAttachedDocumentation(t *testing.T) {
	source := []byte("class Widget {\n  /** Keep this method documented. */\n  private void run() {}\n  void keep() {}\n}\n")
	target := []byte("private void run() {}")
	start := bytes.Index(source, target)
	updated, err := RemoveJavaDeclaration(source, start, start+len(target), target)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(updated), "Keep this method documented") || !strings.Contains(string(updated), "void keep()") {
		t.Fatalf("declaration removal did not preserve the sibling: %s", updated)
	}
	if _, err := RemoveJavaDeclaration(source, start, start+len(target), []byte("private void other() {}")); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("stale source preimage error = %v", err)
	}
}

func TestSetJavaDocCommentUpdatesAndRemovesAttachedComment(t *testing.T) {
	source := []byte("package sample;\nclass Widget {\n  /** Old documentation. */\n  void run() {}\n}\n")
	target := []byte("void run() {}")
	start := bytes.Index(source, target)
	updated, err := SetJavaDocComment(source, start, start+len(target), target, "Runs work.\nSecond line.")
	if err != nil {
		t.Fatal(err)
	}
	want := "package sample;\nclass Widget {\n  /**\n   * Runs work.\n   * Second line.\n   */\n  void run() {}\n}\n"
	if string(updated) != want {
		t.Fatalf("updated source:\n%s\nwant:\n%s", updated, want)
	}
	removed, err := SetJavaDocComment(updated, bytes.Index(updated, target), bytes.Index(updated, target)+len(target), target, "")
	if err != nil {
		t.Fatal(err)
	}
	want = "package sample;\nclass Widget {\n  void run() {}\n}\n"
	if string(removed) != want {
		t.Fatalf("removed source:\n%s\nwant:\n%s", removed, want)
	}
}

func TestSetJavaDocCommentRejectsUnsafeCommentAndWrongRange(t *testing.T) {
	source := []byte("class Widget { void run() {} }\n")
	target := []byte("void run() {}")
	start := bytes.Index(source, target)
	if _, err := SetJavaDocComment(source, start, start+len(target), target, "unsafe */ comment"); err == nil {
		t.Fatal("expected comment terminator to be rejected")
	}
	if _, err := SetJavaDocComment(source, start, start+len(target), []byte("void other() {}"), "docs"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("wrong declaration preimage error = %v, want ErrInvalidPath", err)
	}
}

func TestRemoveJavaDeclarationSupportsMemberKinds(t *testing.T) {
	tests := []struct {
		name   string
		source string
		target string
	}{
		{name: "constructor", source: "class Widget {\n  Widget() {}\n}\n", target: "Widget() {}"},
		{name: "field", source: "class Widget {\n  private int value;\n}\n", target: "private int value;"},
		{name: "nested type", source: "class Widget {\n  record Key(int value) {}\n}\n", target: "record Key(int value) {}"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := []byte(test.source)
			start := strings.Index(test.source, test.target)
			updated, err := RemoveJavaDeclaration(source, start, start+len(test.target), []byte(test.target))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(updated), test.target) || !strings.Contains(string(updated), "class Widget") {
				t.Fatalf("removed source = %s", updated)
			}
			if err := ValidateDeclarationKind([]byte(test.target), map[string]backend.ConstructKind{"constructor": backend.ConstructConstructor, "field": backend.ConstructField, "nested type": backend.ConstructType}[test.name]); err != nil {
				t.Fatalf("validate %s kind: %v", test.name, err)
			}
		})
	}
	if err := ValidateDeclarationKind([]byte("void run() {}"), backend.ConstructField); err == nil {
		t.Fatal("mismatched Java declaration kind unexpectedly succeeded")
	}
}

func TestInsertJavaFieldOrdersMembersAndValidatesSource(t *testing.T) {
	owner := []byte("class Widget {\n  int size;\n  void run() {}\n}\n")
	updated, err := InsertJavaField(owner, []byte("private int timeout;"), "timeout", "type_end", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "class Widget {\n  int size;\n  private int timeout;\n  void run() {}\n}\n"
	if string(updated) != want {
		t.Fatalf("field insertion = %q, want %q", updated, want)
	}
	updated, err = InsertJavaField([]byte("class Widget {}\n"), []byte("int timeout;"), "timeout", "type_start", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSource(updated); err != nil || !strings.Contains(string(updated), "int timeout;") {
		t.Fatalf("empty-type insertion = %q, validation error = %v", updated, err)
	}
	if _, err := InsertJavaField(owner, []byte("int size;"), "size", "type_end", ""); err == nil {
		t.Fatal("duplicate Java field insertion unexpectedly succeeded")
	}
	if _, err := InsertJavaField(owner, []byte("int other;"), "timeout", "type_end", ""); err == nil {
		t.Fatal("mismatched Java field name unexpectedly succeeded")
	}
}

func TestInsertJavaMethodOrdersAndAcceptsConstructor(t *testing.T) {
	owner := []byte("class Widget {\n  private int size;\n  void run() {}\n  class Nested {}\n}\n")
	updated, err := InsertJavaMember(owner, []byte("public Widget() {}"), "Widget", "type_end", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(updated), "private int size;") > strings.Index(string(updated), "public Widget()") || strings.Index(string(updated), "public Widget()") > strings.Index(string(updated), "void run()") || strings.Index(string(updated), "public Widget()") > strings.Index(string(updated), "class Nested") {
		t.Fatalf("constructor insertion order = %s", updated)
	}
	updated, err = InsertJavaMember(owner, []byte("public void close() {}"), "close", "type_end", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(updated), "void close()") > strings.Index(string(updated), "class Nested") {
		t.Fatalf("method was not inserted before nested type: %s", updated)
	}
	if _, insertErr := InsertJavaMember(owner, []byte("private int close;"), "close", "type_end", ""); insertErr == nil {
		t.Fatal("non-method insertion unexpectedly succeeded")
	}
	updated, err = InsertJavaMember(owner, []byte("private static class Helper {}"), "Helper", "type_end", "")
	if err != nil {
		t.Fatalf("insert nested type: %v", err)
	}
	if !strings.Contains(string(updated), "private static class Helper") {
		t.Fatalf("nested type missing from result:\n%s", updated)
	}
}

func TestExtractedLocalIdentifierMatchesInitializer(t *testing.T) {
	source := []byte("class Widget { void run() { int generated = left + right; use(generated); } }")
	name, start, end, err := ExtractedLocalIdentifier(source, []byte("left + right"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "generated" || string(source[start:end]) != name {
		t.Fatalf("local = %q at [%d,%d), want generated identifier", name, start, end)
	}
	if _, _, _, err := ExtractedLocalIdentifier([]byte("class W { void m() { int a = x + y; int b = x + y; } }"), []byte("x + y")); err == nil {
		t.Fatal("ambiguous initializer resolved to one local")
	}
}

func TestJavaLocalDeclarationFindsExactDeclaration(t *testing.T) {
	source := []byte("class W { void run() {\n  int first = 1;\n  String second = \"x\";\n} }")
	candidate, err := JavaLocalDeclaration(source, "second")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(candidate.Syntax, "String second =") || candidate.Range.Start.Line != 2 {
		t.Fatalf("candidate = %#v", candidate)
	}
	if _, err := JavaLocalDeclaration(source, "missing"); err == nil {
		t.Fatal("missing local unexpectedly resolved")
	}
}

func TestExtractedMethodIdentifierMatchesSelectedBody(t *testing.T) {
	source := []byte("class Example { void original() { run(); }\nprivate void extracted() {\n  int total = 1 + 2;\n  consume(total);\n}\n}")
	name, start, end, err := ExtractedMethodIdentifier(source, []byte("int total = 1 + 2;\n  consume(total);"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "extracted" || string(source[start:end]) != name {
		t.Fatalf("extracted method = %q at %q", name, source[start:end])
	}
	if _, _, _, err := ExtractedMethodIdentifier(source, []byte("int missing = 3;")); err == nil {
		t.Fatal("unmatched selected body unexpectedly resolved")
	}
}

func TestJavaStatementRangeSelectsContiguousStatements(t *testing.T) {
	source := []byte("class Example { void run() {\n  int total = 1 + 2;\n  consume(total);\n  finish();\n} }")
	candidate, err := JavaStatementRange(source, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Syntax != "int total = 1 + 2;\n  consume(total);" || candidate.Range.Start.Line != 1 || candidate.Range.End.Line != 2 {
		t.Fatalf("candidate = %#v", candidate)
	}
	if _, err := JavaStatementRange(source, 0, 2); err == nil {
		t.Fatal("range including a method signature unexpectedly resolved")
	}
}

func TestAddJavaImplementedInterfacePreservesExistingAndAppends(t *testing.T) {
	source := []byte("package sample;\nclass Widget extends Base implements Closeable { }\n")
	updated, changed, err := AddJavaImplementedInterface(source, "Widget", "sample.Auditable")
	if err != nil {
		t.Fatal(err)
	}
	if !changed || !strings.Contains(string(updated), "implements Closeable, sample.Auditable") {
		t.Fatalf("updated Java declaration = %q, changed = %v", updated, changed)
	}
	unchanged, changed, err := AddJavaImplementedInterface(updated, "Widget", "sample.Auditable")
	if err != nil {
		t.Fatal(err)
	}
	if changed || !bytes.Equal(unchanged, updated) {
		t.Fatalf("duplicate Java interface changed source: %q", unchanged)
	}
}

func TestJavaInterfaceMethodSignatureAndAppend(t *testing.T) {
	source := []byte("public class Worker { public int size(String label) { return label.length(); } }\n")
	method := []byte("public int size(String label) { return label.length(); }")
	signature, err := JavaInterfaceMethodSignature(method)
	if err != nil {
		t.Fatal(err)
	}
	if signature != "public int size(String label);" {
		t.Fatalf("signature = %q", signature)
	}
	declaration := []byte("interface Measurable {\n\t" + signature + "\n}")
	updated, err := AppendJavaInterface(source, declaration, "Measurable")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSource(updated); err != nil {
		t.Fatalf("appended source is invalid: %v", err)
	}
	if _, err := AppendJavaInterface(updated, declaration, "Measurable"); err == nil {
		t.Fatal("duplicate interface name was accepted")
	}
	if _, err := JavaInterfaceMethodSignature([]byte("public static int size() { return 1; }")); err == nil {
		t.Fatal("static method was accepted as an abstract interface member")
	}
}

func TestInsertJavaSwitchCaseUsesExactSwitchAndPlacement(t *testing.T) {
	method := []byte("void run(int status) {\n\tswitch (status) {\n\tcase 1:\n\t\tbreak;\n\tdefault:\n\t\tbreak;\n\t}\n}")
	updated, err := InsertJavaSwitchCase(method, "status", "", []byte("case 2:\n\tbreak;"), "before_default", "")
	if err != nil {
		t.Fatalf("InsertJavaSwitchCase(): %v", err)
	}
	want := "\tcase 1:\n\t\tbreak;\n\tcase 2:\n\tbreak;\n\tdefault:"
	if !strings.Contains(string(updated), want) {
		t.Fatalf("inserted source does not place case before default:\n%s", updated)
	}
	if _, err := InsertJavaSwitchCase(method, "status", "", []byte("case 1: break;"), "last", ""); err == nil {
		t.Fatal("InsertJavaSwitchCase() accepted a duplicate label")
	}
	if _, err := InsertJavaSwitchCase(method, "status", "", []byte("case 3: break;"), "before", "missing"); err == nil {
		t.Fatal("InsertJavaSwitchCase() accepted a missing target case")
	}
}

func TestIsPrivateJavaMethod(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		private bool
		wantErr bool
	}{
		{name: "private", source: "private void persist() {}", private: true},
		{name: "annotation containing private", source: "@SuppressWarnings(\"private\") void persist() {}"},
		{name: "package private", source: "void persist() {}"},
		{name: "public", source: "public void persist() {}"},
		{name: "field", source: "private int value;", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsPrivateJavaMethod([]byte(tt.source))
			if (err != nil) != tt.wantErr || got != tt.private {
				t.Fatalf("IsPrivateJavaMethod(%q) = %t, %v; want %t, error %t", tt.source, got, err, tt.private, tt.wantErr)
			}
		})
	}
}

func TestReplaceMethodBodyPreservesSignatureAndRejectsInvalidSource(t *testing.T) {
	method := []byte("public int sum(int left, int right) { return left + right; }")
	updated, err := ReplaceMethodBody(method, []byte("return left - right;"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "public int sum(int left, int right)") || !strings.Contains(string(updated), "return left - right;") {
		t.Fatalf("replacement = %s", updated)
	}
	if _, err := ReplaceMethodBody(method, []byte("return ; invalid")); err == nil {
		t.Fatal("invalid Java replacement unexpectedly succeeded")
	}
	if _, err := ReplaceMethodBody([]byte("abstract void run();"), []byte("return;")); err == nil {
		t.Fatal("body replacement unexpectedly accepted an abstract method")
	}
}

func TestIsPrivateJavaDeclaration(t *testing.T) {
	tests := []struct {
		name   string
		source string
		kind   backend.ConstructKind
		want   bool
	}{
		{name: "private method", source: "private void run() {}", kind: backend.ConstructMethod, want: true},
		{name: "package private method", source: "void run() {}", kind: backend.ConstructMethod},
		{name: "annotation text is not visibility", source: "@SuppressWarnings(\"private\") void run() {}", kind: backend.ConstructMethod},
		{name: "private constructor", source: "private __Semedit() {}", kind: backend.ConstructConstructor, want: true},
		{name: "private field", source: "private int value;", kind: backend.ConstructField, want: true},
		{name: "private nested type", source: "private class Nested {}", kind: backend.ConstructType, want: true},
		{name: "public nested type", source: "public interface Nested {}", kind: backend.ConstructType},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := IsPrivateJavaDeclaration([]byte(test.source), test.kind)
			if err != nil || got != test.want {
				t.Fatalf("IsPrivateJavaDeclaration(%q, %q) = %t, %v; want %t", test.source, test.kind, got, err, test.want)
			}
		})
	}
	if _, err := IsPrivateJavaDeclaration([]byte("private int value;"), backend.ConstructMethod); err == nil {
		t.Fatal("IsPrivateJavaDeclaration accepted a declaration with the wrong kind")
	}
}

func TestIsPublicTopLevelType(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		typeName string
		want     bool
	}{
		{name: "public class", source: "package sample; public class Widget {}", typeName: "Widget", want: true},
		{name: "nested public class", source: "package sample; class Container { public static class Widget {} }", typeName: "Widget"},
		{name: "package-private class", source: "package sample; class Widget {}", typeName: "Widget"},
		{name: "multiple top-level declarations", source: "package sample; public class Widget {} class Helper {}", typeName: "Widget"},
		{name: "mismatched name", source: "package sample; public class Widget {}", typeName: "Other"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := IsPublicTopLevelType([]byte(test.source), test.typeName)
			if err != nil {
				t.Fatalf("IsPublicTopLevelType() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("IsPublicTopLevelType() = %v, want %v", got, test.want)
			}
		})
	}
}

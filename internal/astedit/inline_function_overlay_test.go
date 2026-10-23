// WHY: Verifies compiler-backed inline-call planning for concrete Go receiver methods.
package astedit_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/astedit"
)

func TestPlanInlineFunctionOverlaySupportsConcreteMethod(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	before := "package sample\ntype Counter struct{}\nfunc (*Counter) Value() int { return 7 }\nfunc Run(counter *Counter) int { return counter.Value() }\n"
	writeInlineMethodFixture(t, root, map[string]string{
		"go.mod":  "module example.com/inlinemethod\n\ngo 1.24\n",
		"main.go": before,
	})
	plan, err := astedit.PlanInlineFunctionOverlay(context.Background(), root, "main.go", "Counter.Value", astedit.InlineFunctionOptions{
		CallerFile: "main.go", InFunction: "Run", Discriminator: "counter.Value()",
	})
	if err != nil {
		t.Fatalf("PlanInlineFunctionOverlay() error = %v", err)
	}
	want := "package sample\ntype Counter struct{}\nfunc (*Counter) Value() int { return 7 }\nfunc Run(counter *Counter) int { return int(7) }\n"
	if string(plan.Change.After) != want {
		t.Fatalf("planned source = %q, want %q", plan.Change.After, want)
	}
	assertInlineMethodFilesUnchanged(t, root, map[string]string{"main.go": before})
}

func TestPlanInlineFunctionOverlaySelectsMultipleDirectAssignments(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	caller := "package sample\nfunc adjust() float64 { return 1 }\nfunc Run(flag bool) any {\n\tvar second float64\n\tfirst := adjust()\n\tsecond = adjust()\n\tthird := adjust()\n\tif flag { return first + second }\n\treturn third\n}\n"
	writeInlineMethodFixture(t, root, map[string]string{
		"go.mod":  "module example.com/inlinemulticall\n\ngo 1.24\n",
		"main.go": caller,
	})
	revision := sha256.Sum256([]byte(caller))
	plan, err := astedit.PlanInlineFunctionOverlay(context.Background(), root, "main.go", "adjust", astedit.InlineFunctionOptions{
		CallerFile: "main.go", InFunction: "Run", Discriminator: "adjust()", OccurrenceIndices: []int{0, 1}, CallerRevision: hex.EncodeToString(revision[:]),
	})
	if err != nil {
		t.Fatalf("PlanInlineFunctionOverlay() error = %v", err)
	}
	want := "package sample\nfunc adjust() float64 { return 1 }\nfunc Run(flag bool) any {\n\tvar second float64\n\tfirst := float64(1)\n\tsecond = float64(1)\n\tthird := adjust()\n\tif flag { return first + second }\n\treturn third\n}\n"
	if string(plan.Change.After) != want {
		t.Fatalf("planned source = %q, want %q", plan.Change.After, want)
	}
	assertInlineMethodFilesUnchanged(t, root, map[string]string{"main.go": caller})
}

func TestPlanInlineFunctionOverlayRejectsInvalidOccurrenceSelection(t *testing.T) {
	t.Parallel()
	caller := "package sample\nfunc adjust() int { return 1 }\nfunc Run() int { first := adjust(); second := adjust(); return first + second }\n"
	revision := sha256.Sum256([]byte(caller))
	base := astedit.InlineFunctionOptions{CallerFile: "main.go", InFunction: "Run", Discriminator: "adjust()", CallerRevision: hex.EncodeToString(revision[:])}
	cases := []struct {
		name     string
		indices  []int
		revision string
		want     string
	}{
		{name: "duplicate", indices: []int{0, 0}, revision: base.CallerRevision, want: "unique"},
		{name: "descending", indices: []int{1, 0}, revision: base.CallerRevision, want: "strictly increasing"},
		{name: "out of range", indices: []int{0, 2}, revision: base.CallerRevision, want: "out-of-range"},
		{name: "stale", indices: []int{0, 1}, revision: strings.Repeat("0", 64), want: "does not match captured caller source"},
		{name: "missing revision", indices: []int{0, 1}, want: "expected_revision is required"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeInlineMethodFixture(t, root, map[string]string{"go.mod": "module example.com/inlinebadselection\n\ngo 1.24\n", "main.go": caller})
			opts := base
			opts.OccurrenceIndices = test.indices
			opts.CallerRevision = test.revision
			_, err := astedit.PlanInlineFunctionOverlay(context.Background(), root, "main.go", "adjust", opts)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("PlanInlineFunctionOverlay() error = %v, want substring %q", err, test.want)
			}
			assertInlineMethodFilesUnchanged(t, root, map[string]string{"main.go": caller})
		})
	}
}

func TestPlanInlineFunctionOverlayPreservesSingleReturnSelectionWithNestedMatch(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	caller := "package sample\nfunc adjust() int { return 1 }\nfunc Run() int {\n\t_ = func() int { return adjust() }\n\treturn adjust()\n}\n"
	writeInlineMethodFixture(t, root, map[string]string{"go.mod": "module example.com/inlinesinglelegacy\n\ngo 1.24\n", "main.go": caller})
	plan, err := astedit.PlanInlineFunctionOverlay(context.Background(), root, "main.go", "adjust", astedit.InlineFunctionOptions{
		CallerFile: "main.go", InFunction: "Run", Discriminator: "adjust()",
	})
	if err != nil {
		t.Fatalf("PlanInlineFunctionOverlay() error = %v", err)
	}
	want := "package sample\nfunc adjust() int { return 1 }\nfunc Run() int {\n\t_ = func() int { return adjust() }\n\treturn int(1)\n}\n"
	if string(plan.Change.After) != want {
		t.Fatalf("planned source = %q, want %q", plan.Change.After, want)
	}
	assertInlineMethodFilesUnchanged(t, root, map[string]string{"main.go": caller})
}

func TestPlanInlineFunctionOverlayRejectsWrongCalleeAndUnsupportedContext(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, caller, discriminator, want string
	}{
		{name: "wrong callee", caller: "package sample\nfunc adjust() int { return 1 }\nfunc other() int { return 2 }\nfunc Run() int { first := other(); second := other(); return first + second }\n", discriminator: "other()", want: "does not resolve to callee"},
		{name: "nested call context", caller: "package sample\nfunc adjust() int { return 1 }\nfunc wrap(value int) int { return value }\nfunc Run() int { first := wrap(adjust()); second := wrap(adjust()); return first + second }\n", discriminator: "adjust()", want: "not a supported top-level"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeInlineMethodFixture(t, root, map[string]string{"go.mod": "module example.com/inlinecontext\n\ngo 1.24\n", "main.go": test.caller})
			revision := sha256.Sum256([]byte(test.caller))
			_, err := astedit.PlanInlineFunctionOverlay(context.Background(), root, "main.go", "adjust", astedit.InlineFunctionOptions{
				CallerFile: "main.go", InFunction: "Run", Discriminator: test.discriminator, OccurrenceIndices: []int{0, 1}, CallerRevision: hex.EncodeToString(revision[:]),
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("PlanInlineFunctionOverlay() error = %v, want substring %q", err, test.want)
			}
			assertInlineMethodFilesUnchanged(t, root, map[string]string{"main.go": test.caller})
		})
	}
}

func TestPlanInlineFunctionOverlayRejectsShadowedConversionAcrossSites(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	caller := "package sample\nfunc adjust() int { return 1 }\nfunc Run(int func(int) int) int { first := adjust(); second := adjust(); return first + second }\n"
	writeInlineMethodFixture(t, root, map[string]string{"go.mod": "module example.com/inlineshadow\n\ngo 1.24\n", "main.go": caller})
	revision := sha256.Sum256([]byte(caller))
	_, err := astedit.PlanInlineFunctionOverlay(context.Background(), root, "main.go", "adjust", astedit.InlineFunctionOptions{
		CallerFile: "main.go", InFunction: "Run", Discriminator: "adjust()", OccurrenceIndices: []int{0, 1}, CallerRevision: hex.EncodeToString(revision[:]),
	})
	if err == nil || !strings.Contains(err.Error(), "does not preserve the callee result type") {
		t.Fatalf("PlanInlineFunctionOverlay() error = %v, want shadowed conversion refusal", err)
	}
	assertInlineMethodFilesUnchanged(t, root, map[string]string{"main.go": caller})
}

func TestPlanInlineFunctionOverlayRejectsCommentOverlapAfterLineDirective(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	caller := "package sample\nfunc adjust() int { return 1 }\nfunc Run() int {\n//line virtual.go:10\n\treturn adjust(/* keep this comment */)\n}\n"
	writeInlineMethodFixture(t, root, map[string]string{"go.mod": "module example.com/inlinecommentdirective\n\ngo 1.24\n", "main.go": caller})
	_, err := astedit.PlanInlineFunctionOverlay(context.Background(), root, "main.go", "adjust", astedit.InlineFunctionOptions{
		CallerFile: "main.go", InFunction: "Run", Discriminator: "adjust(/* keep this comment */)",
	})
	if err == nil || !strings.Contains(err.Error(), "overlaps a comment or directive") {
		t.Fatalf("PlanInlineFunctionOverlay() error = %v, want comment-overlap refusal", err)
	}
	assertInlineMethodFilesUnchanged(t, root, map[string]string{"main.go": caller})
}

func TestPlanInlineFunctionOverlayRefusesUnsafeMethodCases(t *testing.T) {
	t.Parallel()
	cases := []inlineMethodRefusalCase{
		{
			name: "effectful receiver", symbol: "Counter.Value",
			caller:        "package sample\nfunc Run() int { return newCounter().Value() }\nfunc newCounter() *Counter { return &Counter{} }\n",
			discriminator: "newCounter().Value()",
			wantError:     "inline method call requires a direct caller-parameter receiver",
			callee:        "package sample\ntype Counter struct{}\nfunc (*Counter) Value() int { return 7 }\n",
		},
		{
			name: "implicit receiver dereference", symbol: "ValueCounter.Value",
			caller:        "package sample\nfunc Run(counter *ValueCounter) int { return counter.Value() }\n",
			discriminator: "counter.Value()",
			wantError:     "inline method call requires a direct caller-parameter receiver",
			callee:        "package sample\ntype ValueCounter int\nfunc (ValueCounter) Value() int { return 9 }\n",
		},
		{
			name: "shadowed cross-file result conversion", symbol: "Counter.Value",
			caller:        "package sample\nfunc Run(counter *Counter, int func(int) int) any { return counter.Value() }\n",
			discriminator: "counter.Value()",
			wantError:     "inlined conversion does not preserve the callee result type",
			callee:        "package sample\ntype Counter struct{}\nfunc (*Counter) Value() int { return 7 }\n",
			calleeFile:    "callee.go", callerFile: "caller.go",
		},
		{
			name: "stale cross-file caller", symbol: "Counter.Value",
			caller:        "package sample\nfunc Run(counter *Counter) int { return counter.Value() }\n",
			discriminator: "counter.Value()", callerRevision: strings.Repeat("0", 64),
			wantError:  "call_site.expected_revision does not match captured caller source",
			callee:     "package sample\ntype Counter struct{}\nfunc (*Counter) Value() int { return 7 }\n",
			calleeFile: "callee.go", callerFile: "caller.go",
		},
		{
			name: "stale cross-file callee", symbol: "Counter.Value",
			caller:        "package sample\nfunc Run(counter *Counter) int { return counter.Value() }\n",
			discriminator: "counter.Value()", calleeRevision: strings.Repeat("0", 64),
			wantError:  "expected_revision does not match captured callee source",
			callee:     "package sample\ntype Counter struct{}\nfunc (*Counter) Value() int { return 7 }\n",
			calleeFile: "callee.go", callerFile: "caller.go",
		},
		{
			name: "named method result", symbol: "Counter.Value",
			caller:        "package sample\nfunc Run(counter *Counter) int { return counter.Value() }\n",
			discriminator: "counter.Value()", wantError: "inline methods require one unnamed result",
			callee: "package sample\ntype Counter struct{}\nfunc (*Counter) Value() (result int) { return 7 }\n",
		},
		{
			name: "bodyless method declaration", symbol: "Counter.Value",
			caller:        "package sample\nfunc Run(counter *Counter) int { return counter.Value() }\n",
			discriminator: "counter.Value()", wantError: "inline callee declaration has no body",
			callee: "package sample\ntype Counter struct{}\nfunc (*Counter) Value() int\n",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assertInlineMethodRefusal(t, test)
		})
	}
}

func writeInlineMethodFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func assertInlineMethodFilesUnchanged(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Fatalf("refusal changed %s: got %q, want %q", name, got, want)
		}
	}
}

func TestPlanInlineFunctionOverlayPreservesDeclaredResultType(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	before := "package sample\nfunc adjust() float64 { return 1 }\nfunc Run() any { return adjust() }\n"
	writeInlineMethodFixture(t, root, map[string]string{
		"go.mod":  "module example.com/inlinefunction\n\ngo 1.24\n",
		"main.go": before,
	})
	plan, err := astedit.PlanInlineFunctionOverlay(context.Background(), root, "main.go", "adjust", astedit.InlineFunctionOptions{
		CallerFile: "main.go", InFunction: "Run", Discriminator: "adjust()",
	})
	if err != nil {
		t.Fatalf("PlanInlineFunctionOverlay() error = %v", err)
	}
	want := "package sample\nfunc adjust() float64 { return 1 }\nfunc Run() any { return float64(1) }\n"
	if string(plan.Change.After) != want {
		t.Fatalf("planned source = %q, want %q", plan.Change.After, want)
	}
	assertInlineMethodFilesUnchanged(t, root, map[string]string{"main.go": before})
}

func TestPlanInlineFunctionOverlaySupportsCrossFileMethod(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	files := map[string]string{
		"go.mod":     "module example.com/inlinemethodcrossfile\n\ngo 1.24\n",
		"callee.go":  "package sample\ntype Counter struct{}\nfunc (*Counter) Value() float64 { return 7 }\n",
		"caller.go":  "package sample\nfunc Run(counter *Counter) any { return counter.Value() }\n",
		"sibling.go": "package sample\n// Keep these bytes.\nvar untouched =    1\n",
	}
	writeInlineMethodFixture(t, root, files)
	plan, err := astedit.PlanInlineFunctionOverlay(context.Background(), root, "callee.go", "Counter.Value", astedit.InlineFunctionOptions{
		CallerFile: "caller.go", InFunction: "Run", Discriminator: "counter.Value()",
	})
	if err != nil {
		t.Fatalf("PlanInlineFunctionOverlay() error = %v", err)
	}
	if plan.Change.Path != filepath.Join(root, "caller.go") {
		t.Errorf("changed path = %q, want caller.go", plan.Change.Path)
	}
	want := "package sample\nfunc Run(counter *Counter) any { return float64(7) }\n"
	if string(plan.Change.After) != want {
		t.Errorf("planned source = %q, want %q", plan.Change.After, want)
	}
	for _, name := range []string{"callee.go", "caller.go", "sibling.go"} {
		if string(plan.ReadSet[filepath.Join(root, name)]) != files[name] {
			t.Errorf("captured source for %s does not match the original", name)
		}
	}
	assertInlineMethodFilesUnchanged(t, root, files)
}

type inlineMethodRefusalCase struct {
	name, symbol, caller, discriminator, wantError string
	callee, calleeFile, callerFile                 string
	callerRevision, calleeRevision                 string
}

func assertInlineMethodRefusal(t *testing.T, test inlineMethodRefusalCase) {
	t.Helper()
	root := t.TempDir()
	calleeFile, callerFile := test.calleeFile, test.callerFile
	if calleeFile == "" {
		calleeFile, callerFile = "main.go", "main.go"
	}
	files := map[string]string{"go.mod": "module example.com/inlinemethodrefusal\n\ngo 1.24\n"}
	if calleeFile == callerFile {
		files[calleeFile] = test.callee + strings.TrimPrefix(test.caller, "package sample\n")
	} else {
		files[calleeFile], files[callerFile] = test.callee, test.caller
	}
	writeInlineMethodFixture(t, root, files)
	_, err := astedit.PlanInlineFunctionOverlay(context.Background(), root, calleeFile, test.symbol, astedit.InlineFunctionOptions{
		CallerFile: callerFile, InFunction: "Run", Discriminator: test.discriminator,
		CallerRevision: test.callerRevision, CalleeRevision: test.calleeRevision,
	})
	if err == nil || !strings.Contains(err.Error(), test.wantError) {
		t.Fatalf("PlanInlineFunctionOverlay() error = %v, want diagnostic containing %q", err, test.wantError)
	}
	assertInlineMethodFilesUnchanged(t, root, files)
}

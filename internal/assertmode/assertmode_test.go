// Package assertmode_test verifies failure-mode rewriting preserves safety invariants.
package assertmode_test

import (
	"strings"
	"testing"

	"semedit/internal/assertmode"
)

func TestApply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mode      assertmode.Mode
		src       string
		wantSwaps int
		wantSkips int
		wantFrag  string
	}{
		{
			name: "relax swaps value assertion",
			mode: assertmode.ModeRelax,
			src: `package x

import "testing"

func TestA(t *testing.T) {
	if err := fail(); err != nil {
		t.Fatalf("failed: %v", err)
	}
}
`,
			wantSwaps: 1,
			wantFrag:  `t.Errorf("failed: %v", err)`,
		},
		{
			name: "relax keeps fatal precondition",
			mode: assertmode.ModeRelax,
			src: `package x

import (
	"os"
	"testing"
)

func TestA(t *testing.T) {
	data, err := os.ReadFile("fixture.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if len(data) == 0 {
		t.Errorf("empty fixture")
	}
}
`,
			wantSwaps: 0,
			wantSkips: 1,
		},
		{
			name: "relax skips non-testing receiver",
			mode: assertmode.ModeRelax,
			src: `package x

type Logger struct{}

func (l *Logger) Fatalf(format string, args ...any) {}

func TestA(t *testing.T) {
	log := &Logger{}
	if true {
		log.Fatalf("nope")
	}
}
`,
			wantSwaps: 0,
			wantSkips: 1,
		},
		{
			name: "relax tolerates errors.Is below",
			mode: assertmode.ModeRelax,
			src: `package x

import (
	"errors"
	"testing"
)

var ErrGone = errors.New("gone")

func TestA(t *testing.T) {
	err := fail()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrGone) {
		t.Errorf("wrong error: %v", err)
	}
}
`,
			wantSwaps: 1,
			wantFrag:  `t.Errorf("expected error, got nil")`,
		},
		{
			name: "relax skips else branch",
			mode: assertmode.ModeRelax,
			src: `package x

import "testing"

func TestA(t *testing.T) {
	if ok {
		t.Log("fine")
	} else {
		t.Fatalf("bad")
	}
}
`,
			wantSwaps: 0,
			wantSkips: 1,
		},
		{
			name: "restrict converts continuing assertion",
			mode: assertmode.ModeRestrict,
			src: `package x

import "testing"

func TestA(t *testing.T) {
	if err := fail(); err != nil {
		t.Errorf("failed: %v", err)
	}
}
`,
			wantSwaps: 1,
			wantFrag:  `t.Fatalf("failed: %v", err)`,
		},
	}

	for _, tt := range tests {
		out, res, err := assertmode.Apply("case_test.go", []byte(tt.src), tt.mode)
		if err != nil {
			t.Errorf("%s: Apply failed: %v", tt.name, err)
		}
		if res.Swapped != tt.wantSwaps {
			t.Errorf("%s: swapped = %d, want %d (%v)", tt.name, res.Swapped, tt.wantSwaps, res.Sites)
		}
		skips := len(res.Sites) - res.Swapped
		if skips != tt.wantSkips {
			t.Errorf("%s: skipped = %d, want %d (%v)", tt.name, skips, tt.wantSkips, res.Sites)
		}
		if tt.wantFrag != "" && !strings.Contains(string(out), tt.wantFrag) {
			t.Errorf("%s: output missing %q:\n%s", tt.name, tt.wantFrag, out)
		}
	}
}

func TestApplyRejectsUnknownMode(t *testing.T) {
	t.Parallel()

	src := "package x\n"
	if _, _, err := assertmode.Apply("case_test.go", []byte(src), assertmode.Mode("sideways")); err == nil {
		t.Errorf("expected error for unknown mode, got nil")
	}
}

func TestApplyRejectsBrokenSyntax(t *testing.T) {
	t.Parallel()

	if _, _, err := assertmode.Apply("case_test.go", []byte("package x\nfunc broken( {\n"), assertmode.ModeRelax); err == nil {
		t.Errorf("expected parse error, got nil")
	}
}

// --- Scope-aware sibling propagation tests ---

func TestRelaxSiblingTaintSameScope(t *testing.T) {
	// resp, err := foo() at function level; err is guarded.
	// resp is also tainted because it's a sibling.
	// resp is used unsafely below (field access), so skip.
	t.Parallel()

	src := `package x

import "testing"

func TestA(t *testing.T) {
	resp, err := getResp()
	if err != nil {
		t.Fatalf("getResp: %v", err)
	}
	_ = resp.Data
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if res.Swapped != 0 {
		t.Errorf("swapped = %d, want 0 (resp is used unsafely below)", res.Swapped)
	}
	if len(res.Sites) != 1 {
		t.Fatalf("expected 1 site, got %d: %v", len(res.Sites), res.Sites)
	}
	if res.Sites[0].Action != assertmode.ActionSkippedUnsafe {
		t.Errorf("action = %q, want %q", res.Sites[0].Action, assertmode.ActionSkippedUnsafe)
	}
}

func TestRelaxSiblingTaintSameScopeSafeBelow(t *testing.T) {
	// resp, err := foo() at function level; err is guarded.
	// resp is also tainted. resp is used safely below (only in testing methods), so swap.
	t.Parallel()

	src := `package x

import "testing"

func TestA(t *testing.T) {
	resp, err := getResp()
	if err != nil {
		t.Fatalf("getResp: %v", err)
	}
	t.Logf("resp: %+v", resp)
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if res.Swapped != 1 {
		t.Errorf("swapped = %d, want 1 (resp used only in safe testing method)", res.Swapped)
	}
	if len(res.Sites) != 1 {
		t.Fatalf("expected 1 site, got %d: %v", len(res.Sites), res.Sites)
	}
	if res.Sites[0].Action != assertmode.ActionSwapped {
		t.Errorf("action = %q, want %q", res.Sites[0].Action, assertmode.ActionSwapped)
	}
}

func TestRelaxShortVarDeclDifferentScope(t *testing.T) {
	// Outer: resp, err := foo() at function level
	// Inner: if err := bar(); err != nil { t.Fatalf(...) }
	// The inner err is a DIFFERENT variable (short var decl in if-init).
	// The inner err should NOT be tainted by the outer sibling group.
	// The inner t.Fatalf should be swapped because the inner err is not used unsafely below.
	t.Parallel()

	src := `package x

import "testing"

func TestA(t *testing.T) {
	resp, err := foo()
	if err != nil {
		t.Fatalf("foo: %v", err)
	}
	_ = resp

	if err := bar(); err != nil {
		t.Fatalf("bar: %v", err)
	}
	t.Log("done")
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// First t.Fatalf (foo): skipped because resp is used unsafely below
	// Second t.Fatalf (bar): swapped because inner err is not used unsafely below
	if res.Swapped != 1 {
		t.Errorf("swapped = %d, want 1", res.Swapped)
	}
	if len(res.Sites) != 2 {
		t.Fatalf("expected 2 sites, got %d: %v", len(res.Sites), res.Sites)
	}
	var foundSwapped, foundSkipped bool
	for _, s := range res.Sites {
		switch s.Action {
		case assertmode.ActionSwapped:
			foundSwapped = true
		case assertmode.ActionSkippedUnsafe:
			foundSkipped = true
		}
	}
	if !foundSwapped {
		t.Error("expected one swapped site (inner err)")
	}
	if !foundSkipped {
		t.Error("expected one skipped site (outer resp tainted by sibling)")
	}
}

func TestRelaxNestedIfShortVarDecl(t *testing.T) {
	// Multiple short variable declarations at different scopes.
	// Each err is a different *types.Var.
	t.Parallel()

	src := `package x

import "testing"

func TestA(t *testing.T) {
	if err := foo(); err != nil {
		t.Fatalf("foo: %v", err)
	}
	if err := bar(); err != nil {
		t.Fatalf("bar: %v", err)
	}
	t.Log("done")
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// Both should be swapped - each err is scoped to its if and not used unsafely below.
	if res.Swapped != 2 {
		t.Errorf("swapped = %d, want 2", res.Swapped)
	}
	if len(res.Sites) != 2 {
		t.Fatalf("expected 2 sites, got %d: %v", len(res.Sites), res.Sites)
	}
}

func TestRelaxSiblingInSameScopeNoUnsafeBelow(t *testing.T) {
	// resp, err := foo() at function level.
	// err is guarded. resp is tainted.
	// resp is used safely below (nil comparison), so swap.
	t.Parallel()

	src := `package x

import "testing"

func TestA(t *testing.T) {
	resp, err := getResp()
	if err != nil {
		t.Fatalf("getResp: %v", err)
	}
	if resp == nil {
		t.Errorf("resp was nil")
	}
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// resp is used in a safe position (nil comparison), so swap.
	if res.Swapped != 1 {
		t.Errorf("swapped = %d, want 1", res.Swapped)
	}
}

func TestRelaxForLoopSiblingTaint(t *testing.T) {
	// data, err := os.ReadDir() at function level.
	// err is guarded. data is tainted.
	// data is used unsafely below (range over it), so skip.
	t.Parallel()

	src := `package x

import (
	"os"
	"testing"
)

func TestA(t *testing.T) {
	data, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, f := range data {
		t.Log(f.Name())
	}
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// data is used unsafely below (range iteration), so skip.
	if res.Swapped != 0 {
		t.Errorf("swapped = %d, want 0 (data used unsafely below)", res.Swapped)
	}
}

func TestRelaxRangeLoopVarsNotTainted(t *testing.T) {
	// Range loop variables should not be tainted.
	// t.Fatalf inside a range loop row should be swapped
	// because the loop variable rebinds each iteration.
	t.Parallel()

	src := `package x

import "testing"

func TestA(t *testing.T) {
	for i, v := range items {
		if v == nil {
			t.Fatalf("item %d is nil", i)
		}
		_ = v.Field
	}
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// i and v are range loop variables, excluded from taint.
	// The t.Fatalf is guarded by if v == nil, but v is a loop variable.
	// Since loop variables are excluded, guardVars returns empty, and the call is swapped.
	if res.Swapped != 1 {
		t.Errorf("swapped = %d, want 1 (loop vars excluded from taint)", res.Swapped)
	}
}

func TestRelaxFmtSafeBelow(t *testing.T) {
	// err is guarded. err is used in fmt.Sprintf below (safe).
	t.Parallel()

	src := `package x

import (
	"fmt"
	"testing"
)

func TestA(t *testing.T) {
	err := fail()
	if err != nil {
		t.Fatalf("fail: %v", err)
	}
	msg := fmt.Sprintf("error: %v", err)
	t.Log(msg)
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if res.Swapped != 1 {
		t.Errorf("swapped = %d, want 1 (err used safely in fmt)", res.Swapped)
	}
}

func TestRelaxErrorsAsSafeBelow(t *testing.T) {
	// err is guarded. err is used in errors.As below (safe).
	t.Parallel()

	src := `package x

import (
	"errors"
	"testing"
)

type MyError struct{}

func (e *MyError) Error() string { return "my" }

func TestA(t *testing.T) {
	err := fail()
	if err != nil {
		t.Fatalf("fail: %v", err)
	}
	var me *MyError
	if errors.As(err, &me) {
		t.Logf("my error: %v", me)
	}
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if res.Swapped != 1 {
		t.Errorf("swapped = %d, want 1 (err used safely in errors.As)", res.Swapped)
	}
}

func TestRestrictSkipsElseBranch(t *testing.T) {
	// Restrict mode does not check else-branch membership (Direction C
	// guard-clause restructuring is out of scope).  It swaps freely.
	t.Parallel()

	src := `package x

import "testing"

func TestA(t *testing.T) {
	if ok {
		t.Error("not ok")
	} else {
		t.Fatal("bad")
	}
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRestrict)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// restrict swaps without else-branch filtering (Direction C out of scope)
	if res.Swapped != 1 {
		t.Errorf("swapped = %d, want 1 (restrict swaps freely)", res.Swapped)
	}
}

func TestRelaxGuardedValueDereferencedBelow(t *testing.T) {
	// err is guarded. A value derived from err is dereferenced below.
	// The sibling from the same assignment is also tainted.
	t.Parallel()

	src := `package x

import "testing"

func TestA(t *testing.T) {
	client, err := newClient()
	if err != nil {
		t.Fatalf("newClient: %v", err)
	}
	resp := client.Do()
	_ = resp.Body
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// client is tainted (sibling of err). client.Do() is an unsafe use.
	if res.Swapped != 0 {
		t.Errorf("swapped = %d, want 0 (client used unsafely below)", res.Swapped)
	}
}

func TestRelaxMultipleGuardedCalls(t *testing.T) {
	// Multiple t.Fatalf in the same function, some safe to swap, some not.
	t.Parallel()

	src := `package x

import (
	"os"
	"testing"
)

func TestA(t *testing.T) {
	data, err := os.ReadFile("a.txt")
	if err != nil {
		t.Fatalf("read a: %v", err)
	}
	_ = data

	content, err := os.ReadFile("b.txt")
	if err != nil {
		t.Fatalf("read b: %v", err)
	}
	t.Logf("content: %s", content)
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// First: skipped (data used unsafely below)
	// Second: swapped (content used safely below)
	if res.Swapped != 1 {
		t.Errorf("swapped = %d, want 1", res.Swapped)
	}
	if len(res.Sites) != 2 {
		t.Fatalf("expected 2 sites, got %d: %v", len(res.Sites), res.Sites)
	}
}

func TestRelaxSwitchCaseGuarded(t *testing.T) {
	// t.Fatalf inside a switch case body.
	t.Parallel()

	src := `package x

import "testing"

func TestA(t *testing.T) {
	switch err := getErr(); err {
	case nil:
		t.Fatalf("expected error")
	default:
		t.Logf("got error: %v", err)
	}
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// err is from the switch init. err is used in the default case (safe - logging).
	// The t.Fatalf is guarded by the switch, and err is only used safely below.
	if res.Swapped != 1 {
		t.Errorf("swapped = %d, want 1 (err used safely in switch)", res.Swapped)
	}
}

func TestRelaxNestedBlockShortVarDecl(t *testing.T) {
	// Short variable declaration in a nested block.
	// The inner err should not affect the outer scope.
	t.Parallel()

	src := `package x

import "testing"

func TestA(t *testing.T) {
	if true {
		if err := inner(); err != nil {
			t.Fatalf("inner: %v", err)
		}
	}
	t.Log("done")
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if res.Swapped != 1 {
		t.Errorf("swapped = %d, want 1 (inner err scoped to inner block)", res.Swapped)
	}
}

func TestRelaxStraightLineNoGuard(t *testing.T) {
	// t.Fatalf without a guarding if. In relax mode, this should be skipped
	// because the call-argument variables are tainted.
	t.Parallel()

	src := `package x

import "testing"

func TestA(t *testing.T) {
	err := fail()
	t.Fatalf("fail: %v", err)
	t.Log("after")
}
`
	_, res, err := assertmode.Apply("test.go", []byte(src), assertmode.ModeRelax)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// err is a call argument, so it's tainted.
	// err is used below in t.Log (safe - testing method).
	// So the t.Fatalf should be swapped.
	if res.Swapped != 1 {
		t.Errorf("swapped = %d, want 1 (err used safely below)", res.Swapped)
	}
}

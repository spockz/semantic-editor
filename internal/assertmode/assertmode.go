// Package assertmode rewrites Go test assertion failure modes between fail-fast and continue-on-failure.
//
// The engine exists to apply ADR-0033 mechanically: abort-on-failure calls
// (t.Fatalf, t.Fatal) become continuing calls (t.Errorf, t.Error) wherever
// the test remains executable after the failure, and vice versa for the
// restrict direction. Receiver types are verified with go/types so rewrites
// never touch same-named methods on other types; guarded values referenced
// below the failure site are reported for manual restructuring instead of
// being bare-swapped into a panic.
package assertmode

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
)

// Mode selects the rewrite direction.
type Mode string

const (
	// ModeRelax converts fail-fast assertions to continue-on-failure assertions.
	ModeRelax Mode = "relax"
	// ModeRestrict converts continue-on-failure assertions to fail-fast assertions.
	ModeRestrict Mode = "restrict"
)

// pairs maps an assertion method to its counterpart in the given direction.
// Pairs share identical signatures so argument lists transfer untouched.
var pairs = map[Mode]map[string]string{
	ModeRelax: {
		"Fatal":  "Error",
		"Fatalf": "Errorf",
	},
	ModeRestrict: {
		"Error":  "Fatal",
		"Errorf": "Fatalf",
	},
}

// Action records what happened to one candidate call site.
type Action string

const (
	// ActionSwapped marks a rewritten call.
	ActionSwapped Action = "swapped"
	// ActionSkippedUnverified marks a call whose receiver type is unavailable or not *testing.T/B.
	ActionSkippedUnverified Action = "skipped-unverified-receiver"
	// ActionSkippedUnsafe marks a call whose guarded value is referenced unsafely below.
	ActionSkippedUnsafe Action = "skipped-unsafe-continuation"
	// ActionSkippedElse marks a call inside an else branch, which inversion cannot preserve.
	ActionSkippedElse Action = "skipped-else-branch"
)

// Site describes one candidate call and its outcome.
type Site struct {
	Line   int    // 1-based source line of the call.
	Call   string // Receiver and method as written, e.g. "t.Fatalf".
	Action Action
	Detail string
}

// Result summarizes one rewritten file.
type Result struct {
	File    string
	Mode    Mode
	Sites   []Site
	Swapped int
}

// siblingGroup holds variables assigned together in one statement: when a
// guard variable fails, its assignment siblings hold unusable values too.
type siblingGroup struct {
	vars []*types.Var
}

// Apply rewrites assertion calls in src according to mode, returning the new
// source and a per-site report. Sites that cannot be proven safe keep their
// original text and appear as skipped. Apply never writes to disk.
func Apply(filename string, src []byte, mode Mode) ([]byte, Result, error) {
	pair, ok := pairs[mode]
	if !ok {
		return nil, Result{}, fmt.Errorf("assertmode: unknown mode %q", mode)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		return nil, Result{}, fmt.Errorf("assertmode: parse %s: %w", filename, err)
	}
	info := &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{Importer: importer.Default(), Error: func(error) {}}
	_, _ = conf.Check("assertmode", fset, []*ast.File{file}, info)

	analyzer := &analyzer{fset: fset, info: info, pair: pair, mode: mode}
	analyzer.siblings = assignmentSiblings(fset, file, info)
	analyzer.file(file)

	out := splice(src, analyzer.splices)
	if _, err := parser.ParseFile(token.NewFileSet(), filename, out, 0); err != nil {
		return nil, Result{}, fmt.Errorf("assertmode: rewrite of %s broke syntax: %w", filename, err)
	}
	if formatted, err := format.Source(out); err != nil || !bytes.Equal(formatted, out) {
		return nil, Result{}, fmt.Errorf("assertmode: rewrite of %s broke gofmt cleanliness", filename)
	}
	swapped := 0
	for _, site := range analyzer.sites {
		if site.Action == ActionSwapped {
			swapped++
		}
	}
	return out, Result{File: filename, Mode: mode, Sites: analyzer.sites, Swapped: swapped}, nil
}

// splice replaces the half-open byte ranges with replacement text.
// Ranges must be disjoint; they apply from the end so offsets stay valid.
func splice(src []byte, edits []spliceEdit) []byte {
	out := append([]byte(nil), src...)
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		out = append(out[:edit.start], append([]byte(edit.text), out[edit.end:]...)...)
	}
	return out
}

type spliceEdit struct {
	start int
	end   int
	text  string
}

type analyzer struct {
	fset    *token.FileSet
	info    *types.Info
	pair    map[string]string
	mode    Mode
	sites   []Site
	splices []spliceEdit
	// siblings groups variables assigned together in one statement, scoped by
	// depth: when a guard variable fails, only siblings at the same scope
	// depth hold unusable values too.
	siblings []siblingGroup
}

// qualifies reports whether obj is a *testing.T or *testing.B variable,
// the only receiver types whose Fatal/Error methods form valid pairs.
func (a *analyzer) qualifies(obj types.Object) bool {
	variable, ok := obj.(*types.Var)
	if !ok {
		return false
	}
	pointer, ok := variable.Type().(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := pointer.Elem().(*types.Named)
	if !ok {
		return false
	}
	name := named.Obj().Name()
	if name != "T" && name != "B" {
		return false
	}
	pkg := named.Obj().Pkg()
	return pkg != nil && pkg.Path() == "testing"
}

func (a *analyzer) file(file *ast.File) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		a.block(fn.Body.List, nil, nil, nil)
	}
}

// ifContext describes a candidate call nested in an if statement body.
type ifContext struct {
	stmt     *ast.IfStmt // Innermost if whose then-branch contains the call.
	elseBody bool        // Call sits in an else branch rather than the then branch.
}

// block walks statements, tracking the innermost guarding if, else-branch
// membership, and range-loop variables rebound each iteration. outer carries
// the continuation: statements executing after the current list completes,
// so uses below a nested guard see past their enclosing statement.
func (a *analyzer) block(stmts []ast.Stmt, guard *ifContext, loopVars map[*types.Var]bool, outer []ast.Stmt) {
	for i, stmt := range stmts {
		rest := append(append([]ast.Stmt(nil), stmts[i+1:]...), outer...)
		switch typed := stmt.(type) {
		case *ast.IfStmt:
			a.ifStmt(typed, guard, loopVars, rest)
		case *ast.ForStmt:
			a.block(typed.Body.List, guard, mergeLoopVars(loopVars, a.rangeVars(typed)), rest)
		case *ast.RangeStmt:
			a.block(typed.Body.List, guard, mergeLoopVars(loopVars, a.rangeVars(typed)), rest)
		case *ast.ExprStmt:
			if call, ok := typed.X.(*ast.CallExpr); ok {
				a.call(call, guard, rest, loopVars)
			}
		default:
			a.nested(stmt, guard, loopVars, rest)
		}
	}
}

// ifStmt analyzes the branches of one if statement. Else-if chains recurse
// as ordinary if statements; plain else blocks mark membership for skipping.
func (a *analyzer) ifStmt(stmt *ast.IfStmt, guard *ifContext, loopVars map[*types.Var]bool, rest []ast.Stmt) {
	a.block(stmt.Body.List, &ifContext{stmt: stmt}, loopVars, rest)
	switch els := stmt.Else.(type) {
	case *ast.BlockStmt:
		a.block(els.List, &ifContext{stmt: stmt, elseBody: true}, loopVars, rest)
	case *ast.IfStmt:
		a.ifStmt(els, guard, loopVars, rest)
	}
}

// nested descends into switch/select cases and function literals, inheriting
// the current guard context. Continuation past case bodies is approximated by
// the outer continuation; case-local ordering is preserved.
func (a *analyzer) nested(stmt ast.Stmt, guard *ifContext, loopVars map[*types.Var]bool, outer []ast.Stmt) {
	bodies := func(list []ast.Stmt) {
		for i, item := range list {
			switch clause := item.(type) {
			case *ast.CaseClause:
				a.block(clause.Body, guard, loopVars, append(append([]ast.Stmt(nil), list[i+1:]...), outer...))
			case *ast.CommClause:
				a.block(clause.Body, guard, loopVars, append(append([]ast.Stmt(nil), list[i+1:]...), outer...))
			}
		}
	}
	switch typed := stmt.(type) {
	case *ast.SwitchStmt:
		bodies(typed.Body.List)
	case *ast.TypeSwitchStmt:
		bodies(typed.Body.List)
	case *ast.SelectStmt:
		bodies(typed.Body.List)
	}
	ast.Inspect(stmt, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.CaseClause, *ast.CommClause:
			return false // Bodies already covered above.
		}
		if lit, ok := node.(*ast.FuncLit); ok && lit.Body != nil {
			a.block(lit.Body.List, guard, loopVars, outer)
			return false
		}
		return true
	})
}

// rangeVars collects variables rebound by for and range headers.
func (a *analyzer) rangeVars(stmt ast.Stmt) map[*types.Var]bool {
	vars := map[*types.Var]bool{}
	collect := func(id *ast.Ident) {
		if obj, ok := a.info.Defs[id]; ok {
			if v, ok := obj.(*types.Var); ok {
				vars[v] = true
			}
		}
	}
	switch typed := stmt.(type) {
	case *ast.ForStmt:
		if typed.Init != nil {
			ast.Inspect(typed.Init, func(node ast.Node) bool {
				if id, ok := node.(*ast.Ident); ok {
					collect(id)
				}
				return true
			})
		}
	case *ast.RangeStmt:
		for _, expr := range []ast.Expr{typed.Key, typed.Value} {
			if id, ok := expr.(*ast.Ident); ok && id.Name != "_" {
				collect(id)
			}
		}
	}
	return vars
}

func mergeLoopVars(base, extra map[*types.Var]bool) map[*types.Var]bool {
	merged := make(map[*types.Var]bool, len(base)+len(extra))
	for v := range base {
		merged[v] = true
	}
	for v := range extra {
		merged[v] = true
	}
	return merged
}

// call evaluates one call expression as a rewrite candidate.
func (a *analyzer) call(call *ast.CallExpr, guard *ifContext, rest []ast.Stmt, loopVars map[*types.Var]bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	recv, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}
	if _, ok := a.pair[sel.Sel.Name]; !ok {
		return
	}
	line := a.fset.Position(sel.Pos()).Line
	name := recv.Name + "." + sel.Sel.Name
	skip := func(action Action, detail string) {
		a.sites = append(a.sites, Site{Line: line, Call: name, Action: action, Detail: detail})
	}
	obj := a.info.Uses[recv]
	if obj == nil {
		obj = a.info.Defs[recv]
	}
	if !a.qualifies(obj) {
		skip(ActionSkippedUnverified, "receiver is not *testing.T or *testing.B")
		return
	}
	if a.mode == ModeRestrict {
		a.swap(sel, line, name)
		return
	}
	if guard != nil && guard.elseBody {
		skip(ActionSkippedElse, "call sits in an else branch")
		return
	}
	tainted := expandTaint(a.guardVars(call, guard, loopVars), a.siblings)
	if a.usedUnsafely(tainted, guard, rest) {
		skip(ActionSkippedUnsafe, "guarded value is referenced below the failure site")
		return
	}
	a.swap(sel, line, name)
}

// swap records one method-name replacement as a byte splice over the
// selector identifier only; the receiver and argument list are untouched.
func (a *analyzer) swap(sel *ast.SelectorExpr, line int, name string) {
	start := a.fset.Position(sel.Sel.Pos()).Offset
	end := a.fset.Position(sel.Sel.End()).Offset
	a.splices = append(a.splices, spliceEdit{start: start, end: end, text: a.pair[sel.Sel.Name]})
	a.sites = append(a.sites, Site{Line: line, Call: name, Action: ActionSwapped})
}

// assignmentSiblings groups variables assigned together in one statement.
// A failure that poisons one group member poisons the rest: continuing past
// `data, err := os.ReadFile(name)` with a failed err leaves data unusable.
func assignmentSiblings(fset *token.FileSet, file *ast.File, info *types.Info) []siblingGroup {
	_ = fset
	var groups []siblingGroup
	add := func(ids []*ast.Ident) {
		var group []*types.Var
		for _, id := range ids {
			if id == nil || id.Name == "_" {
				continue
			}
			obj := info.Uses[id]
			if obj == nil {
				obj = info.Defs[id]
			}
			if v, ok := obj.(*types.Var); ok {
				group = append(group, v)
			}
		}
		if len(group) > 1 {
			groups = append(groups, siblingGroup{vars: group})
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			var ids []*ast.Ident
			for _, lhs := range typed.Lhs {
				if id, ok := lhs.(*ast.Ident); ok {
					ids = append(ids, id)
				}
			}
			add(ids)
		case *ast.ValueSpec:
			add(typed.Names)
		}
		return true
	})
	return groups
}

// expandTaint grows the tainted set through assignment siblings to a fixpoint,
// so values derived alongside a failed guard variable stay tainted.
func expandTaint(tainted map[*types.Var]bool, siblings []siblingGroup) map[*types.Var]bool {
	for changed := true; changed; {
		changed = false
		for _, g := range siblings {
			hit := false
			for _, v := range g.vars {
				if tainted[v] {
					hit = true
					break
				}
			}
			if !hit {
				continue
			}
			for _, v := range g.vars {
				if !tainted[v] {
					tainted[v] = true
					changed = true
				}
			}
		}
	}
	return tainted
}

// guardVars collects the variables tainted by the guard: condition and init
// variables for if-guarded calls, call-argument variables for straight-line
// calls. Range-loop variables are excluded; they rebind every iteration.
func (a *analyzer) guardVars(call *ast.CallExpr, guard *ifContext, loopVars map[*types.Var]bool) map[*types.Var]bool {
	vars := map[*types.Var]bool{}
	add := func(id *ast.Ident) {
		obj := a.info.Uses[id]
		if obj == nil {
			obj = a.info.Defs[id]
		}
		if v, ok := obj.(*types.Var); ok && !loopVars[v] {
			vars[v] = true
		}
	}
	if guard == nil {
		for _, arg := range call.Args {
			ast.Inspect(arg, func(node ast.Node) bool {
				if id, ok := node.(*ast.Ident); ok {
					add(id)
				}
				return true
			})
		}
		return vars
	}
	// Collect only variables that are actually tested in the condition.
	// Variables appearing in init expressions but not in the condition are
	// not tainted by the guard (e.g., a file path argument to os.WriteFile).
	condVars := map[*types.Var]bool{}
	ast.Inspect(guard.stmt.Cond, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok {
			obj := a.info.Uses[id]
			if obj == nil {
				obj = a.info.Defs[id]
			}
			if v, ok := obj.(*types.Var); ok && !loopVars[v] {
				condVars[v] = true
			}
		}
		return true
	})
	if guard.stmt.Init != nil {
		ast.Inspect(guard.stmt.Init, func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok {
				obj := a.info.Uses[id]
				if obj == nil {
					obj = a.info.Defs[id]
				}
				if v, ok := obj.(*types.Var); ok && !loopVars[v] && condVars[v] {
					vars[v] = true
				}
			}
			return true
		})
	}
	// Also add any variable directly used in the condition.
	for v := range condVars {
		vars[v] = true
	}
	return vars
}

// usedUnsafely reports whether any tainted variable is referenced below the
// anchor in a position that is unsound on the failure path. References inside
// errors.Is/As/AsType first arguments and direct nil comparisons are sound:
// those functions and comparisons tolerate nil. The anchor's own else branch
// counts as below: it executes on the unguarded path.
func (a *analyzer) usedUnsafely(tainted map[*types.Var]bool, guard *ifContext, rest []ast.Stmt) bool {
	if len(tainted) == 0 {
		return false
	}
	check := func(stmts []ast.Stmt) bool {
		unsafe := false
		for _, stmt := range stmts {
			safe := map[*ast.Ident]bool{}
			ast.Inspect(stmt, func(node ast.Node) bool {
				a.markSafeUses(node, tainted, safe)
				return true
			})
			ast.Inspect(stmt, func(node ast.Node) bool {
				id, ok := node.(*ast.Ident)
				if !ok {
					return true
				}
				obj, ok := a.info.Uses[id].(*types.Var)
				if !ok || !tainted[obj] || safe[id] {
					return true
				}
				unsafe = true
				return false
			})
			if unsafe {
				return true
			}
		}
		return false
	}
	if guard != nil {
		if els, ok := guard.stmt.Else.(*ast.BlockStmt); ok && check(els.List) {
			return true
		}
	}
	return check(rest)
}

// markSafeUses records identifier nodes in sound positions: the exact first
// argument of errors.Is/As/AsType, either side of a direct nil comparison,
// and direct arguments of testing assertion/log methods and fmt formatting
// calls. Formatting verbs never dereference, so message arguments cannot
// panic on the failure path; nested expressions (deref, index, method call)
// stay tracked.
func (a *analyzer) markSafeUses(node ast.Node, tainted map[*types.Var]bool, safe map[*ast.Ident]bool) {
	direct := func(arg ast.Expr) {
		if id, ok := arg.(*ast.Ident); ok {
			if obj, ok := a.info.Uses[id].(*types.Var); ok && tainted[obj] {
				safe[id] = true
			}
		}
	}
	switch typed := node.(type) {
	case *ast.CallExpr:
		sel, ok := typed.Fun.(*ast.SelectorExpr)
		if !ok || len(typed.Args) == 0 {
			return
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok {
			return
		}
		if pkgName, ok := a.info.Uses[pkg].(*types.PkgName); ok {
			switch pkgName.Imported().Path() {
			case "errors":
				switch sel.Sel.Name {
				case "Is", "As", "AsType":
					direct(typed.Args[0])
				}
				return
			case "fmt":
				switch sel.Sel.Name {
				case "Sprint", "Sprintf", "Sprintln", "Print", "Printf", "Println",
					"Fprint", "Fprintf", "Fprintln", "Errorf":
					for _, arg := range typed.Args {
						direct(arg)
					}
				}
				return
			}
		}
		if a.qualifies(a.info.Uses[pkg]) {
			for _, arg := range typed.Args {
				direct(arg)
			}
		}
	case *ast.BinaryExpr:
		if typed.Op != token.EQL && typed.Op != token.NEQ {
			return
		}
		for _, side := range []ast.Expr{typed.X, typed.Y} {
			id, ok := side.(*ast.Ident)
			if !ok {
				continue
			}
			other := typed.Y
			if side == typed.Y {
				other = typed.X
			}
			nilIdent, ok := other.(*ast.Ident)
			if !ok || nilIdent.Name != "nil" {
				continue
			}
			if obj, ok := a.info.Uses[id].(*types.Var); ok && tainted[obj] {
				safe[id] = true
			}
		}
	}
}

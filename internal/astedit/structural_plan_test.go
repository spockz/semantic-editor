// This file checks source-planner outcomes and refusals without exercising runtime publication.

package astedit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestStructuralSourcePlanners(t *testing.T) {
	t.Run("field insertion preserves the selected struct and exposes semantic gates", assertFieldInsertionPreservesSemanticGates)
	t.Run("field insertion refuses positional literals and direct collisions", assertFieldInsertionRefusals)
	t.Run("field insertion uses the exact brace span for empty structs", assertEmptyStructFieldInsertion)
}

func assertFieldInsertionPreservesSemanticGates(t *testing.T) {
	source := []byte("package p\ntype config struct {\n\t// Current value.\n\tcurrent int `json:\"current\"`\n}\n")
	plan, err := PlanInsertField(source, InsertFieldIntent{TypeName: "config", FieldName: "added", FieldSource: "added string `json:\"added\"`", Placement: FieldBeforeSymbol, TargetField: "current"})
	if err != nil {
		t.Fatal(err)
	}
	expected := "package p\ntype config struct {\n\tadded string `json:\"added\"`\n\t// Current value.\n\tcurrent int `json:\"current\"`\n}\n"
	if string(plan.Source) != expected {
		t.Fatalf("planned source mismatch:\n%s", plan.Source)
	}
	if !plan.Changed || plan.Selection.Target != "config.added" {
		t.Fatalf("missing changed target evidence: %+v", plan)
	}
	if err := typeCheckPlannedSource(t, plan.Source); err != nil {
		t.Fatalf("planned field source did not type-check: %v", err)
	}
	if len(plan.RequiredProofs) != 3 || plan.RequiredProofs[0] != ProofCompleteSourceSet || plan.RequiredProofs[2] != ProofUnkeyedLiteralSet {
		t.Fatalf("missing integration proof gates: %+v", plan.RequiredProofs)
	}
}

func assertFieldInsertionRefusals(t *testing.T) {
	for _, source := range []string{"package p\ntype config struct {\n\tcurrent int\n}\nvar value = config{1}\n", "package p\ntype config struct {\n\tadded int\n}\n"} {
		_, err := PlanInsertField([]byte(source), InsertFieldIntent{TypeName: "config", FieldName: "added", FieldSource: "added string"})
		if err == nil {
			t.Fatalf("expected refusal for %q", source)
		}
	}
	for _, fieldSource := range []string{"_ int", "added int\n}\ntype injected int\ntype holder struct {"} {
		_, err := PlanInsertField([]byte("package p\ntype config struct {\n\tcurrent int\n}\n"), InsertFieldIntent{TypeName: "config", FieldName: "added", FieldSource: fieldSource})
		if err == nil {
			t.Fatalf("unsafe field source %q was accepted", fieldSource)
		}
	}
}

func assertEmptyStructFieldInsertion(t *testing.T) {
	cases := []struct {
		name, source, expected string
	}{
		{name: "empty struct", source: "package p\ntype options struct{}\n", expected: "package p\ntype options struct{\n\ttimeout int\n}\n"},
		{name: "grouped type with indentation", source: "package p\ntype (\n\toptions struct{}\n)\n", expected: "package p\ntype (\n\toptions struct{\n\t\ttimeout int\n\t}\n)\n"},
		{name: "CRLF", source: "package p\r\ntype options struct{}\r\n", expected: "package p\r\ntype options struct{\r\n\ttimeout int\r\n}\r\n"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			plan, err := PlanInsertField([]byte(testCase.source), InsertFieldIntent{TypeName: "options", FieldName: "timeout", FieldSource: "timeout int"})
			if err != nil {
				t.Fatal(err)
			}
			if string(plan.Source) != testCase.expected {
				t.Fatalf("empty struct edit = %q, want %q", plan.Source, testCase.expected)
			}
			if err := typeCheckPlannedSource(t, plan.Source); err != nil {
				t.Fatalf("empty struct result did not type-check: %v", err)
			}
		})
	}
}

func TestStructuralStatementPlannerAnchors(t *testing.T) {
	t.Run("statement insertion uses exact direct anchors in control flow owners", func(t *testing.T) {
		source := []byte("package p\nfunc Handle() {\n\tfirst()\n\tsecond()\n}\n")
		plan, err := PlanInsertStatement(source, InsertStatementIntent{
			Function: "Handle", Discriminator: "second()", Source: "audit()",
			Placement: StatementBefore,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(plan.Source), "\taudit()\n\tsecond()") || plan.Selection.Kind != "statement" || len(plan.RequiredProofs) != 3 {
			t.Fatalf("unexpected statement plan: %+v\n%s", plan, plan.Source)
		}
		controlFlow := []byte("package p\nfunc Handle() {\n\tif ready {\n\t\tfirst()\n\t}\n\tsecond()\n}\n")
		controlPlan, err := PlanInsertStatement(controlFlow, InsertStatementIntent{
			Function: "Handle", Discriminator: "second()", ConstructPath: OptionalString{Present: true, Value: "1"}, ExpectedRevision: OptionalString{Present: true, Value: goReadRevision(controlFlow)}, Source: "audit()", Placement: StatementBefore,
		})
		if err != nil || !strings.Contains(string(controlPlan.Source), "audit()") {
			t.Fatalf("control-flow owner direct anchor failed: plan=%+v err=%v", controlPlan, err)
		}
		for _, source := range []string{
			"package p\nfunc Handle() { x(); y() }\n",
			"package p\nfunc Handle() {\n\tx() /* inline */ y()\n}\n",
		} {
			_, err := PlanInsertStatement([]byte(source), InsertStatementIntent{
				Function: "Handle", Discriminator: "x()", Source: "audit()", Placement: StatementBefore,
			})
			if err == nil {
				t.Fatalf("shared-line statement source was accepted: %s", source)
			}
		}
	})
}

func TestStructuralMethodSelectorPlanner(t *testing.T) {
	t.Run("qualified method selectors preserve receiver identity", func(t *testing.T) {
		source := []byte("package p\ntype A struct{}\ntype B struct{}\nfunc (A) Run() {\n\twork()\n}\nfunc (B) Run() {\n\twork()\n}\n")
		plan, err := PlanInsertStatement(source, InsertStatementIntent{
			Function: "A.Run", Discriminator: "work()", Source: "audit()", Placement: StatementBefore,
		})
		if err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, "out.go", plan.Source, parser.ParseComments|parser.AllErrors)
		if err != nil {
			t.Fatalf("planned source did not parse: %v", err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil {
				continue
			}
			gotAudit := len(fn.Body.List) > 0
			if gotAudit {
				expr, ok := fn.Body.List[0].(*ast.ExprStmt)
				if ok {
					call, callOK := expr.X.(*ast.CallExpr)
					ident, identOK := astNodeName(call, callOK)
					gotAudit = identOK && ident == "audit"
				} else {
					gotAudit = false
				}
			}
			if (fn.Recv.List[0].Type.(*ast.Ident).Name == "A") != gotAudit {
				t.Fatalf("qualified selector edited wrong receiver method %s", fn.Name.Name)
			}
		}
		_, err = PlanInsertStatement(source, InsertStatementIntent{
			Function: "C.Run", Discriminator: "work()", Source: "audit()", Placement: StatementBefore,
		})
		if err == nil {
			t.Fatal("missing qualified receiver unexpectedly selected another method")
		}
	})
}

func TestStructuralTypedMapPlanner(t *testing.T) {
	t.Run("typed map insertion detects semantic duplicate keys", func(t *testing.T) {
		source := []byte("package p\nvar handlers = map[string]int{\"a\": 1}\n")
		intent := InsertElementIntent{
			Declaration: "handlers", Discriminator: "map[string]int{\"a\": 1}", Source: "\"b\": 2",
			TypeContext: CompositeTypeContext{
				Family: LiteralMap, MapKeyType: "string",
				ExistingMapKeys: []TypedKeyEvidence{{Expression: "\"a\"", Identity: "string:a"}},
				NewMapKey:       TypedKeyEvidence{Expression: "\"b\"", Identity: "string:b"},
			},
		}
		plan, err := PlanInsertCompositeElement(source, intent)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(plan.Source), "\"a\": 1, \"b\": 2") || plan.Selection.Target != "" || len(plan.RequiredProofs) != 2 {
			t.Fatalf("unexpected literal plan: %+v\n%s", plan, plan.Source)
		}
		intent.Source = "aliasA: 2"
		intent.TypeContext.NewMapKey = TypedKeyEvidence{Expression: "aliasA", Identity: "string:a"}
		_, err = PlanInsertCompositeElement(source, intent)
		if err == nil {
			t.Fatal("typed duplicate map key was accepted")
		}
		_, _, err = parseKeyedElement("\"b\": 2}, extra int\nvar injected = 1\nvar _ = map[any]any{")
		if err == nil {
			t.Fatal("injected declaration in element source was accepted")
		}
		for _, source := range []string{`"b": 2,`, `"b": 2 /* trailing note */`} {
			if _, _, err := parseKeyedElement(source); err == nil {
				t.Errorf("element source with separator/comment was accepted: %q", source)
			}
		}
	})
}

func TestStructuralDocCommentPlanner(t *testing.T) {
	t.Run("declaration doc comments preserve text and distinguish removal from missing", func(t *testing.T) {
		assertDocCommentPlanBehavior(t)
	})
}

func assertDocCommentPlanBehavior(t *testing.T) {
	t.Helper()
	source := []byte("package p\ntype Settings struct{}\n")
	comment := "Service settings.\nLoaded once."
	plan, err := PlanSetDocComment(source, SetDocCommentIntent{Symbol: "Settings", Comment: OptionalString{Present: true, Value: comment}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plan.Source), "// Service settings.\n// Loaded once.\ntype Settings") {
		t.Fatalf("comment was not set: %s", plan.Source)
	}
	assertDocCommentSet(t, plan.Source)
	removed, err := PlanSetDocComment(plan.Source, SetDocCommentIntent{Symbol: "Settings", Comment: OptionalString{Present: true, Value: ""}})
	if err != nil {
		t.Fatal(err)
	}
	assertDocCommentRemoved(t, removed.Source)
	assertDocCommentRemovalAndDirectiveRefusals(t, source, comment)
}

func assertDocCommentSet(t *testing.T, source []byte) {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), "out.go", source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		t.Fatalf("comment replacement did not parse: %v", err)
	}
	doc := parsed.Decls[0].(*ast.GenDecl).Doc
	if doc == nil || doc.Text() != "Service settings.\nLoaded once.\n" {
		t.Fatalf("replacement comment is not attached to declaration: %#v", doc)
	}
}

func assertDocCommentRemoved(t *testing.T, source []byte) {
	t.Helper()
	if strings.Contains(string(source), "Service settings") {
		t.Fatalf("comment was not removed: %s", source)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), "removed.go", source, parser.ParseComments|parser.AllErrors)
	if err != nil || parsed.Decls[0].(*ast.GenDecl).Doc != nil {
		t.Fatalf("comment removal did not leave declaration doc-free: err=%v", err)
	}
}

func assertDocCommentRemovalAndDirectiveRefusals(t *testing.T, source []byte, comment string) {
	t.Helper()
	_, err := PlanSetDocComment(source, SetDocCommentIntent{Symbol: "Settings", Comment: OptionalString{}})
	if err == nil {
		t.Fatal("missing comment was treated as removal")
	}
	assertDirectiveDocCommentRefusal(t, comment)
	_, err = PlanSetDocComment([]byte("package p\nvar a, b int\n"), SetDocCommentIntent{Symbol: "a", Comment: OptionalString{Present: true, Value: "owned?"}})
	if err == nil {
		t.Fatal("shared ValueSpec comment ownership was accepted")
	}
}

func assertDirectiveDocCommentRefusal(t *testing.T, comment string) {
	t.Helper()
	directive := "package p\n//go:generate tool\ntype Settings struct{}\n"
	for _, replacement := range []string{comment, "new comment", ""} {
		before := []byte(directive)
		_, err := PlanSetDocComment(before, SetDocCommentIntent{Symbol: "Settings", Comment: OptionalString{Present: true, Value: replacement}})
		if err == nil || string(before) != directive {
			t.Fatalf("directive doc accepted replacement %q or mutated input: err=%v", replacement, err)
		}
	}
}

func astNodeName(call *ast.CallExpr, ok bool) (string, bool) {
	if !ok {
		return "", false
	}
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return "", false
	}
	return ident.Name, true
}

func typeCheckPlannedSource(t *testing.T, source []byte) error {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "planned.go", source, parser.AllErrors)
	if err != nil {
		return err
	}
	_, err = (&types.Config{}).Check(file.Name.Name, fset, []*ast.File{file}, nil)
	return err
}

func TestStructCompositeElementTypeContext(t *testing.T) {
	source := []byte("package p\ntype config struct {\n\tName string\n\tEnabled bool\n}\nvar cfg = config{Name: \"service\"}\n")
	intent := InsertElementIntent{
		Declaration: "cfg", Discriminator: "config{Name: \"service\"}", Source: "Enabled: true",
		TypeContext: CompositeTypeContext{
			Family:       LiteralStruct,
			StructFields: []FieldSpec{{Name: "Name", Type: "string"}, {Name: "Enabled", Type: "bool"}},
		},
	}
	plan, err := PlanInsertCompositeElement(source, intent)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plan.Source), "config{Name: \"service\", Enabled: true}") {
		t.Fatalf("struct element was not appended: %s", plan.Source)
	}
	intent.Source = "Missing: true"
	_, err = PlanInsertCompositeElement(source, intent)
	if err == nil {
		t.Fatal("unknown typed struct field was accepted")
	}
	intent.Source = "Name: \"other\""
	_, err = PlanInsertCompositeElement(source, intent)
	if err == nil {
		t.Fatal("duplicate struct field was accepted")
	}
	grouped := []byte("package p\ntype config struct {\n\tName string\n\tEnabled bool\n}\nvar (\n\tfirst = config{Name: \"first\"}\n\tsecond = config{Name: \"second\"}\n)\n")
	groupIntent := InsertElementIntent{
		Declaration: "first", Discriminator: "config{Name: \"first\"}", Source: "Enabled: true",
		TypeContext: CompositeTypeContext{Family: LiteralStruct, StructFields: []FieldSpec{{Name: "Name", Type: "string"}, {Name: "Enabled", Type: "bool"}}},
	}
	groupPlan, err := PlanInsertCompositeElement(grouped, groupIntent)
	if err != nil {
		t.Fatal(err)
	}
	if err := typeCheckPlannedSource(t, groupPlan.Source); err != nil {
		t.Fatalf("grouped declaration insertion did not type-check: %v", err)
	}
	if !strings.Contains(string(groupPlan.Source), "first = config{Name: \"first\", Enabled: true}") || strings.Contains(string(groupPlan.Source), "second = config{Name: \"second\", Enabled: true}") {
		t.Fatalf("composite insertion escaped exact ValueSpec ownership:\n%s", groupPlan.Source)
	}
}

func TestStructuralOptionalSelectorAndCommentInputs(t *testing.T) {
	source := []byte("package p\nfunc Handle() {\n\trun()\n}\n")
	_, err := PlanInsertStatement(source, InsertStatementIntent{
		Function: "Handle", Discriminator: "run()", ConstructPath: OptionalString{Present: true, Null: true},
		ExpectedRevision: OptionalString{Present: true, Value: goReadRevision(source)}, Source: "audit()", Placement: StatementBefore,
	})
	if err == nil {
		t.Fatal("explicit null construct_path was treated as omission")
	}
	_, err = PlanInsertStatement(source, InsertStatementIntent{
		Function: "Handle", Discriminator: "run()", ConstructPath: OptionalString{Present: true, Value: ""},
		ExpectedRevision: OptionalString{Present: true, Value: goReadRevision(source)}, Source: "audit()", Placement: StatementBefore,
	})
	if err == nil {
		t.Fatal("explicit empty construct_path was accepted")
	}
	zeroPath := OptionalString{Present: true, Value: "0"}
	revision := OptionalString{Present: true, Value: goReadRevision(source)}
	_, err = PlanInsertStatement(source, InsertStatementIntent{
		Function: "Handle", Discriminator: "run()", ConstructPath: zeroPath, ExpectedRevision: revision,
		Source: "audit()", Placement: StatementBefore,
	})
	if err != nil {
		t.Fatalf("explicit exact-anchor path should select a live statement: err=%v", err)
	}
	_, err = PlanInsertStatement(source, InsertStatementIntent{
		Function: "Handle", Discriminator: "run()", ExpectedRevision: OptionalString{Present: true, Value: "stale"},
		Source: "audit()", Placement: StatementBefore,
	})
	if err == nil || !strings.Contains(err.Error(), "revision") {
		t.Fatalf("stale source revision was accepted: err=%v", err)
	}
	comment := OptionalString{Present: true, Null: true}
	_, err = PlanSetDocComment([]byte("package p\ntype Settings struct{}\n"), SetDocCommentIntent{Symbol: "Settings", Comment: comment})
	if err == nil {
		t.Fatal("explicit null comment was treated as removal")
	}
}

func TestDocCommentBlockCRLFRawRange(t *testing.T) {
	source := []byte("package p\r\n/* Old\r\nMore\r\n*/\r\nfunc f() {}\r\n")
	set, err := PlanSetDocComment(source, SetDocCommentIntent{
		Symbol: "f", Comment: OptionalString{Present: true, Value: "New documentation."},
	})
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "updated.go", set.Source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		t.Fatalf("updated CRLF block comment source did not parse: %v\n%s", err, set.Source)
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	if fn.Doc == nil || fn.Doc.Text() != "New documentation.\n" || strings.Contains(string(set.Source), "*/") {
		t.Fatalf("replacement comment was not attached or old raw terminator remains:\n%s", set.Source)
	}
	if typeCheckErr := typeCheckPlannedSource(t, set.Source); typeCheckErr != nil {
		t.Fatalf("updated CRLF comment source did not type-check: %v", typeCheckErr)
	}
	empty := OptionalString{Present: true, Value: ""}
	removed, err := PlanSetDocComment(set.Source, SetDocCommentIntent{Symbol: "f", Comment: empty})
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("package p\r\nfunc f() {}\r\n")
	if string(removed.Source) != string(want) {
		t.Fatalf("comment removal changed bytes outside its owned range:\n got %q\nwant %q", removed.Source, want)
	}
}

func TestStructuralMethodSelectorCanonicalizesReceiverTypes(t *testing.T) {
	for _, receiverTypes := range []string{"A[T]", "A[T, U]"} {
		typeParameters := "T any"
		if receiverTypes == "A[T, U]" {
			typeParameters = "T, U any"
		}
		source := []byte("package p\ntype A[" + typeParameters + "] struct{}\nfunc (" + receiverTypes + ") Value() {\n\tvalueWork()\n}\nfunc (*" + receiverTypes + ") Pointer() {\n\tpointerWork()\n}\n")
		for _, testCase := range []struct {
			selector string
			anchor   string
		}{
			{selector: "A.Value", anchor: "valueWork()"},
			{selector: "A.Pointer", anchor: "pointerWork()"},
			{selector: "(*A).Pointer", anchor: "pointerWork()"},
		} {
			plan, err := PlanInsertStatement(source, InsertStatementIntent{
				Function: testCase.selector, Discriminator: testCase.anchor, Source: "audit()", Placement: StatementBefore,
			})
			if err != nil {
				t.Errorf("receiver %s selector %q: %v", receiverTypes, testCase.selector, err)
				continue
			}
			if !strings.Contains(string(plan.Source), "audit()\n\t"+testCase.anchor) {
				t.Errorf("receiver %s selector %q did not insert before %s:\n%s", receiverTypes, testCase.selector, testCase.anchor, plan.Source)
			}
		}
	}
}

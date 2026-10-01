// This file checks source-planner outcomes and refusals without exercising runtime publication.

package astedit

import (
	"strings"
	"testing"
)

func TestStructuralSourcePlanners(t *testing.T) {
	t.Run("field insertion preserves the selected struct and exposes semantic gates", func(t *testing.T) {
		source := []byte("package p\ntype config struct {\n\t// Current value.\n\tcurrent int `json:\"current\"`\n}\n")
		plan, err := PlanInsertField(source, InsertFieldIntent{
			TypeName: "config", FieldName: "added", FieldSource: "added string `json:\"added\"`",
			Placement: FieldBeforeSymbol, TargetField: "current",
		})
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
		if len(plan.RequiredProofs) != 3 || plan.RequiredProofs[0] != ProofCompleteSourceSet || plan.RequiredProofs[2] != ProofUnkeyedLiteralSet {
			t.Fatalf("missing integration proof gates: %+v", plan.RequiredProofs)
		}
	})
	t.Run("field insertion refuses positional literals and direct collisions", func(t *testing.T) {
		for _, source := range []string{
			"package p\ntype config struct {\n\tcurrent int\n}\nvar value = config{1}\n",
			"package p\ntype config struct {\n\tadded int\n}\n",
		} {
			_, err := PlanInsertField([]byte(source), InsertFieldIntent{TypeName: "config", FieldName: "added", FieldSource: "added string"})
			if err == nil {
				t.Fatalf("expected refusal for %q", source)
			}
		}
	})
	t.Run("statement insertion uses an exact direct anchor and refuses control flow", func(t *testing.T) {
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
		_, err = PlanInsertStatement(source, InsertStatementIntent{
			Function: "Handle", Discriminator: "first()", ConstructPath: OptionalString{Present: true, Value: "0"}, Source: "audit()",
			Placement: StatementAfter,
		})
		if err == nil {
			t.Fatal("construct_path without expected_revision was accepted")
		}
		controlFlow := []byte("package p\nfunc Handle() {\n\tif ready {\n\t\tfirst()\n\t}\n}\n")
		_, err = PlanInsertStatement(controlFlow, InsertStatementIntent{
			Function: "Handle", Discriminator: "first()", Source: "audit()", Placement: StatementBefore,
		})
		if err == nil {
			t.Fatal("control-flow owner was accepted")
		}
	})
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
	})
	t.Run("declaration doc comments preserve text and distinguish removal from missing", func(t *testing.T) {
		source := []byte("package p\ntype Settings struct{}\n")
		comment := "Service settings.\nLoaded once."
		plan, err := PlanSetDocComment(source, SetDocCommentIntent{Symbol: "Settings", Comment: OptionalString{Present: true, Value: comment}})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(plan.Source), "// Service settings.\n// Loaded once.\ntype Settings") {
			t.Fatalf("comment was not set: %s", plan.Source)
		}
		empty := ""
		removed, err := PlanSetDocComment(plan.Source, SetDocCommentIntent{Symbol: "Settings", Comment: OptionalString{Present: true, Value: empty}})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(removed.Source), "Service settings") {
			t.Fatalf("comment was not removed: %s", removed.Source)
		}
		_, err = PlanSetDocComment(source, SetDocCommentIntent{Symbol: "Settings", Comment: OptionalString{}})
		if err == nil {
			t.Fatal("missing comment was treated as removal")
		}
		directive := []byte("package p\n//go:generate tool\ntype Settings struct{}\n")
		_, err = PlanSetDocComment(directive, SetDocCommentIntent{Symbol: "Settings", Comment: OptionalString{Present: true, Value: comment}})
		if err == nil {
			t.Fatal("directive-bearing doc comment was accepted")
		}
	})
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
}

func TestStructuralOptionalSelectorAndCommentInputs(t *testing.T) {
	source := []byte("package p\nfunc Handle() {\n\trun()\n}\n")
	_, err := PlanInsertStatement(source, InsertStatementIntent{
		Function: "Handle", Discriminator: "run()", ConstructPath: OptionalString{Present: true, Null: true},
		ExpectedRevision: OptionalString{Present: true, Value: "rev"}, Source: "audit()", Placement: StatementBefore,
	})
	if err == nil {
		t.Fatal("explicit null construct_path was treated as omission")
	}
	_, err = PlanInsertStatement(source, InsertStatementIntent{
		Function: "Handle", Discriminator: "run()", ConstructPath: OptionalString{Present: true, Value: ""},
		ExpectedRevision: OptionalString{Present: true, Value: "rev"}, Source: "audit()", Placement: StatementBefore,
	})
	if err == nil {
		t.Fatal("explicit empty construct_path was accepted")
	}
	zeroPath := OptionalString{Present: true, Value: "0"}
	revision := OptionalString{Present: true, Value: "opaque"}
	plan, err := PlanInsertStatement(source, InsertStatementIntent{
		Function: "Handle", Discriminator: "run()", ConstructPath: zeroPath, ExpectedRevision: revision,
		Source: "audit()", Placement: StatementBefore,
	})
	if err != nil || !plan.Changed {
		t.Fatalf("zero path with its revision should select the statement: plan=%+v err=%v", plan, err)
	}
	comment := OptionalString{Present: true, Null: true}
	_, err = PlanSetDocComment([]byte("package p\ntype Settings struct{}\n"), SetDocCommentIntent{Symbol: "Settings", Comment: comment})
	if err == nil {
		t.Fatal("explicit null comment was treated as removal")
	}
}

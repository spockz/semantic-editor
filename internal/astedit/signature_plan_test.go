// WHY: Proves signature planners preserve declaration, caller, body, and parameter evaluation meaning.

package astedit

import (
	"strings"
	"testing"
)

func TestPlanChangeSignatureUpdatesDeclarationAndEveryProvenCall(t *testing.T) {
	target := "package p\nfunc work(id int) bool { return id > 0 }\n"
	callerA := "package p\nfunc run() bool { return work(1) }\n"
	callerB := "package p\nfunc retry() bool { return work(2) }\n"
	callA := strings.Index(callerA, "work(1)")
	callB := strings.Index(callerB, "work(2)")
	zero := 0
	request := SignatureChangeRequest{
		File: "work.go", Symbol: "work",
		Parameters: []SignatureParameterMapping{
			{SourceIndex: &zero, Name: "id", Type: "int"},
			{Name: "enabled", Type: "bool", Argument: "true"},
		},
	}
	proof := SignatureCallSetProof{
		Complete: true,
		Files: []SignatureSourceFile{
			{File: "work.go", Source: []byte(target)},
			{File: "a.go", Source: []byte(callerA)},
			{File: "b.go", Source: []byte(callerB)},
		},
		Calls: []SignatureCallSite{
			{File: "a.go", Start: callA, End: callA + len("work(1)")},
			{File: "b.go", Start: callB, End: callB + len("work(2)")},
		},
	}
	plan, err := PlanChangeSignature(request, proof)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Files) != 3 {
		t.Fatalf("planned files = %d, want 3", len(plan.Files))
	}
	planned := make(map[string]string, len(plan.Files))
	for _, file := range plan.Files {
		planned[file.File] = string(file.After)
	}
	if !strings.Contains(planned["work.go"], "func work(id int, enabled bool)") {
		t.Fatalf("signature output = %s", planned["work.go"])
	}
	if !strings.Contains(planned["a.go"], "work(1, true)") || !strings.Contains(planned["b.go"], "work(2, true)") {
		t.Fatalf("caller outputs = %#v", planned)
	}
	if planned["a.go"] == callerA || planned["b.go"] == callerB {
		t.Fatal("caller preimages were not changed")
	}
}

func TestPlanChangeSignatureUsesCallerLocalPositionsForZeroArgumentCall(t *testing.T) {
	target := "package p\nfunc work() int { return 1 }\n"
	caller := "package p\nfunc noise() int { return 7 }\nfunc run() int { return work(\n) }\n"
	call := strings.Index(caller, "work(\n)")
	request := SignatureChangeRequest{File: "work.go", Symbol: "work", Parameters: []SignatureParameterMapping{{Name: "enabled", Type: "bool", Argument: "true"}}}
	plan, err := PlanChangeSignature(request, SignatureCallSetProof{
		Complete: true,
		Files:    []SignatureSourceFile{{File: "work.go", Source: []byte(target)}, {File: "caller.go", Source: []byte(caller)}},
		Calls:    []SignatureCallSite{{File: "caller.go", Start: call, End: call + len("work(\n)")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range plan.Files {
		if file.File == "caller.go" {
			if got, want := string(file.After), "package p\nfunc noise() int { return 7 }\nfunc run() int { return work(\ntrue) }\n"; got != want {
				t.Fatalf("caller output = %q, want %q", got, want)
			}
			return
		}
	}
	t.Fatal("plan omitted the cross-file caller")
}

func TestPlanIntroduceParameterReplacesOnlyEquivalentConstant(t *testing.T) {
	target := "package p\nfunc ready(id int) bool { return true }\n"
	caller := "package p\nfunc run() bool { return ready(1) }\n"
	call := strings.Index(caller, "ready(1)")
	zero := 0
	request := IntroduceParameterRequest{
		Signature: SignatureChangeRequest{
			File: "ready.go", Symbol: "ready",
			Parameters: []SignatureParameterMapping{
				{SourceIndex: &zero, Name: "id", Type: "int"},
				{Name: "enabled", Type: "bool", Argument: "true"},
			},
		},
		Expression:    "true",
		ParameterName: "enabled",
		Constant:      SignatureConstantEvidence{Type: "bool", SelectedValue: "bool:true", ArgumentValue: "bool:true", ArgumentSource: "true"},
	}
	plan, err := PlanIntroduceParameter(request, SignatureCallSetProof{
		Complete: true,
		Files:    []SignatureSourceFile{{File: "ready.go", Source: []byte(target)}, {File: "run.go", Source: []byte(caller)}},
		Calls:    []SignatureCallSite{{File: "run.go", Start: call, End: call + len("ready(1)")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	planned := make(map[string]string, len(plan.Files))
	for _, file := range plan.Files {
		planned[file.File] = string(file.After)
	}
	if !strings.Contains(planned["ready.go"], "func ready(id int, enabled bool) bool { return enabled }") {
		t.Fatalf("target output = %s", planned["ready.go"])
	}
	if !strings.Contains(planned["run.go"], "ready(1, true)") {
		t.Fatalf("caller output = %s", planned["run.go"])
	}
	request.Constant.HasSideEffects = true
	if _, err := PlanIntroduceParameter(request, SignatureCallSetProof{Complete: true, Files: []SignatureSourceFile{{File: "ready.go", Source: []byte(target)}}}); err == nil {
		t.Fatal("side-effect timing hazard was accepted")
	}
}

func TestPlanIntroduceParameterObjectPreservesArgumentOrder(t *testing.T) {
	target := "package p\nfunc save(path, body string) string { return path + body }\n"
	caller := "package p\nfunc run() string { return save(first(), second()) }\n"
	call := strings.Index(caller, "save(first(), second())")
	bodyPath := strings.LastIndex(target, "path")
	bodyValue := strings.LastIndex(target, "body")
	request := ParameterObjectRequest{
		File: "save.go", Symbol: "save", ParameterName: "options", TypeName: "saveOptions",
		Fields: []ParameterObjectField{
			{SourceIndex: 0, Name: "path", Type: "string"},
			{SourceIndex: 1, Name: "body", Type: "string"},
		},
	}
	plan, err := PlanIntroduceParameterObject(request, ParameterObjectProof{
		Complete: true,
		Files:    []SignatureSourceFile{{File: "save.go", Source: []byte(target)}, {File: "run.go", Source: []byte(caller)}},
		Calls:    []SignatureCallSite{{File: "run.go", Start: call, End: call + len("save(first(), second())")}},
		BodyBindings: []SignatureBodyBinding{
			{File: "save.go", Start: bodyPath, End: bodyPath + len("path"), SourceIndex: 0},
			{File: "save.go", Start: bodyValue, End: bodyValue + len("body"), SourceIndex: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	planned := make(map[string]string, len(plan.Files))
	for _, file := range plan.Files {
		planned[file.File] = string(file.After)
	}
	if !strings.Contains(planned["save.go"], "func save(options saveOptions) string { return options.path + options.body }") {
		t.Fatalf("target output = %s", planned["save.go"])
	}
	if !strings.Contains(planned["save.go"], "type saveOptions struct") || !strings.Contains(planned["save.go"], "path string") || !strings.Contains(planned["save.go"], "body string") {
		t.Fatalf("object output = %s", planned["save.go"])
	}
	if !strings.Contains(planned["run.go"], "save(saveOptions{path: first(), body: second()})") {
		t.Fatalf("caller output = %s", planned["run.go"])
	}
}

func TestSignaturePlannersRequireExplicitCompleteReadSets(t *testing.T) {
	if _, err := PlanChangeSignature(SignatureChangeRequest{}, SignatureCallSetProof{}); err == nil {
		t.Fatal("incomplete caller-set proof was accepted")
	}
	if _, err := PlanIntroduceParameterObject(ParameterObjectRequest{}, ParameterObjectProof{}); err == nil {
		t.Fatal("incomplete parameter-object proof was accepted")
	}
}

func TestPlanChangeSignatureNoOpTrailingCommaAndVariadicRefusal(t *testing.T) {
	zero := 0
	target := "package p\nfunc work(id int,\n) bool { return id > 0 }\n"
	caller := "package p\nfunc run() bool { return work(1,\n) }\n"
	call := strings.Index(caller, "work(1,")
	proof := SignatureCallSetProof{
		Complete: true,
		Files:    []SignatureSourceFile{{File: "work.go", Source: []byte(target)}, {File: "caller.go", Source: []byte(caller)}},
		Calls:    []SignatureCallSite{{File: "caller.go", Start: call, End: call + len("work(1,\n)")}},
	}
	unchanged := SignatureChangeRequest{File: "work.go", Symbol: "work", Parameters: []SignatureParameterMapping{{SourceIndex: &zero, Name: "id", Type: "int"}}}
	noOp, err := PlanChangeSignature(unchanged, proof)
	if err != nil {
		t.Fatal(err)
	}
	if len(noOp.Files) != 0 {
		t.Fatalf("no-op planned %d file writes, want zero", len(noOp.Files))
	}
	request := SignatureChangeRequest{File: "work.go", Symbol: "work", Parameters: []SignatureParameterMapping{
		{SourceIndex: &zero, Name: "id", Type: "int"},
		{Name: "enabled", Type: "bool", Argument: "true"},
	}}
	plan, err := PlanChangeSignature(request, proof)
	if err != nil {
		t.Fatal(err)
	}
	planned := make(map[string]string, len(plan.Files))
	for _, file := range plan.Files {
		planned[file.File] = string(file.After)
	}
	if !strings.Contains(planned["work.go"], "func work(id int, enabled bool") {
		t.Fatalf("signature output = %s", planned["work.go"])
	}
	if !strings.Contains(planned["caller.go"], "work(1, true") {
		t.Fatalf("caller output = %s", planned["caller.go"])
	}
	variadicTarget := "package p\nfunc work(ids ...int) {}\n"
	variadic := true
	variadicRequest := SignatureChangeRequest{File: "variadic.go", Symbol: "work", Parameters: []SignatureParameterMapping{
		{SourceIndex: &zero, Name: "ids", Type: "int", Variadic: variadic},
		{Name: "enabled", Type: "bool", Argument: "true"},
	}}
	_, err = PlanChangeSignature(variadicRequest, SignatureCallSetProof{Complete: true, Files: []SignatureSourceFile{{File: "variadic.go", Source: []byte(variadicTarget)}}})
	if err == nil || !strings.Contains(err.Error(), "variadic") {
		t.Fatalf("variadic append error = %v, want refusal", err)
	}
}

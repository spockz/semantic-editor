// Package operation tests Scala callable selection boundaries against source text.
// WHY: A resolved Scala symbol range can stop before its full block, so callable selections must remain anchored to that method's parsed brace scope.
package operation

import (
	"strings"
	"testing"
)

func TestScalaCallableSelectionStaysWithinResolvedOwnerBody(t *testing.T) {
	source := []byte(`object A {
  def run(): Unit = /* {{ nested /* } */ */ {
    consume("{")
  }
}
object B {
  def run(): Unit = {
    consume("}")
  }
} // }
`)
	ownerStart := strings.Index(string(source), "def run")
	firstCall := strings.Index(string(source), `consume("{")`)
	secondCall := strings.Index(string(source), `consume("}")`)
	if ownerStart < 0 || firstCall < 0 || secondCall < 0 {
		t.Fatal("test fixture is malformed")
	}
	if !scalaSelectionWithinOwnerBody(source, ownerStart, len(source), firstCall, firstCall+len(`consume("{")`)) {
		t.Fatal("selection inside the resolved owner was rejected")
	}
	if scalaSelectionWithinOwnerBody(source, ownerStart, len(source), secondCall, secondCall+len(`consume("}")`)) {
		t.Fatal("selection in a later same-named method was accepted")
	}

	expressionOwner := []byte("object A { def run(): Unit = consume(0) }\nobject B { def run(): Unit = { consume(1) } }\n")
	expressionOwnerStart := strings.Index(string(expressionOwner), "def run")
	foreignCall := strings.LastIndex(string(expressionOwner), "consume(1)")
	if scalaSelectionWithinOwnerBody(expressionOwner, expressionOwnerStart, len(expressionOwner), foreignCall, foreignCall+len("consume(1)")) {
		t.Fatal("selection in a later method was accepted after an expression-bodied owner")
	}

	interpolatedOwner := []byte("object A { def run(): Unit = { val text = s\"${\"{\"}\"; consume(1) } }\n")
	interpolatedStart := strings.Index(string(interpolatedOwner), "def run")
	interpolatedCall := strings.Index(string(interpolatedOwner), "consume(1)")
	if scalaSelectionWithinOwnerBody(interpolatedOwner, interpolatedStart, len(interpolatedOwner), interpolatedCall, interpolatedCall+len("consume(1)")) {
		t.Fatal("owner with interpolation syntax was accepted by the bounded body scanner")
	}
	for _, prefix := range []string{"s", "custom", "custom$", "custom_+"} {
		owner := []byte("object A { def run(): Unit = { val text = " + prefix + `"${"{"}"; consume(1) } }` + "\n")
		methodStart := strings.Index(string(owner), "def run")
		call := strings.Index(string(owner), "consume(1)")
		if scalaSelectionWithinOwnerBody(owner, methodStart, len(owner), call, call+len("consume(1)")) {
			t.Fatalf("owner with %q interpolation syntax was accepted by the bounded body scanner", prefix)
		}
	}

	symbolOwner := []byte("object A { def run(): Unit = { val first = 'hello; consume(1) } }\nobject B { def run(): Unit = { consume(2); val second = 'world } }\n")
	symbolOwnerStart := strings.Index(string(symbolOwner), "def run")
	symbolCall := strings.Index(string(symbolOwner), "consume(1)")
	if scalaSelectionWithinOwnerBody(symbolOwner, symbolOwnerStart, len(symbolOwner), symbolCall, symbolCall+len("consume(1)")) {
		t.Fatal("owner with Symbol literals was accepted by the bounded body scanner")
	}
	backquotedOwner := []byte("object A { def run(): Unit = { val `x{` = 1; consume(1) } }\nobject B { def run(): Unit = { consume(2) } }\n")
	backquotedStart := strings.Index(string(backquotedOwner), "def run")
	backquotedCall := strings.Index(string(backquotedOwner), "consume(1)")
	if scalaSelectionWithinOwnerBody(backquotedOwner, backquotedStart, len(backquotedOwner), backquotedCall, backquotedCall+len("consume(1)")) {
		t.Fatal("owner with a brace in a backquoted identifier was accepted by the bounded body scanner")
	}
	xmlOwner := []byte("object A { def run(): Unit = { val ignored = <a><!-- { --></a>; consume(1) } }\nobject B { def run(): Unit = { consume(2) } }\n")
	xmlStart := strings.Index(string(xmlOwner), "def run")
	xmlCall := strings.Index(string(xmlOwner), "consume(1)")
	if scalaSelectionWithinOwnerBody(xmlOwner, xmlStart, len(xmlOwner), xmlCall, xmlCall+len("consume(1)")) {
		t.Fatal("owner with XML literal syntax was accepted by the bounded body scanner")
	}
	abstractOwner := []byte("trait A { def run(): Unit }\nobject B { def run(): Unit = { consume(1) } }\n")
	abstractStart := strings.Index(string(abstractOwner), "def run")
	laterOwnerStart := strings.Index(string(abstractOwner), "object B")
	abstractCall := strings.Index(string(abstractOwner), "consume(1)")
	if scalaSelectionWithinOwnerBody(abstractOwner, abstractStart, laterOwnerStart, abstractCall, abstractCall+len("consume(1)")) {
		t.Fatal("abstract owner borrowed the body of a later same-named method")
	}
}

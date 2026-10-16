// Package suggestions tests bounded ranking and candidate retrieval.
package suggestions

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestMatcherRanksExactTranspositionAndDistinctMethods(t *testing.T) {
	candidates := []Candidate{{Name: "ReadConfig", File: "z.go"}, {Name: "ReadConfg", File: "b.go"}, {Name: "ReadConifg", File: "a.go"}}
	got, err := (Matcher{}).Retrieve(context.Background(), Query{Selector: "ReadConfig", Candidates: candidates})
	if err != nil || !reflect.DeepEqual(got, []Candidate{candidates[0], candidates[2], candidates[1]}) {
		t.Fatalf("Retrieve() = %#v, %v", got, err)
	}
	methods := []Candidate{{Selector: "Store.Read", Name: "Read"}, {Selector: "Reader.Read", Name: "Read"}}
	got, err = (Matcher{}).Retrieve(context.Background(), Query{Selector: "Read", Candidates: methods})
	if err != nil || len(got) != 2 || got[0].Selector != "Reader.Read" || got[1].Selector != "Store.Read" {
		t.Fatalf("methods = %#v, %v", got, err)
	}
	got, err = (Matcher{}).Retrieve(context.Background(), Query{Selector: "Cuz", Candidates: []Candidate{{Name: "Cat"}, {Name: "Cut"}}})
	if err != nil || len(got) != 0 {
		t.Fatalf("short query = %#v, %v", got, err)
	}
}

func TestMatcherBoundsResultsAndCancellation(t *testing.T) {
	candidates := make([]Candidate, maximumLimit+3)
	for i := range candidates {
		candidates[i].Name = "Read"
	}
	got, err := (Matcher{}).Retrieve(context.Background(), Query{Selector: "Read", Candidates: candidates, Limit: maximumLimit + 2})
	if err != nil || len(got) != maximumLimit {
		t.Fatalf("got %d, %v", len(got), err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (Matcher{}).Retrieve(ctx, Query{Selector: "Read"}); err == nil {
		t.Fatal("canceled query succeeded")
	}
}

func TestMatcherKeepsJavaOverloadSignaturesForSignatureFreeQueries(t *testing.T) {
	candidates := []Candidate{
		{Selector: "com.example.Widget.parseConfig(int)", Name: "parseConfig", Kind: "method", Language: "java", Identity: "int"},
		{Selector: "com.example.Widget.parseConfig(java.lang.String)", Name: "parseConfig", Kind: "method", Language: "java", Identity: "string"},
	}
	got, err := (Matcher{}).Retrieve(context.Background(), Query{Selector: "Widget.parsConfg", Language: "java", Candidates: candidates})
	if err != nil || len(got) != 2 || got[0].Selector != candidates[0].Selector || got[1].Selector != candidates[1].Selector {
		t.Fatalf("overload retrieval = %#v, %v; want both canonical overloads", got, err)
	}
	got, err = (Matcher{}).Retrieve(context.Background(), Query{Selector: "parsConfg", Language: "java", Candidates: candidates})
	if err != nil || len(got) != 2 {
		t.Fatalf("unqualified overload retrieval = %#v, %v; want both overloads", got, err)
	}
}

func TestMatcherUsesRustPathSeparatorForLeafMatches(t *testing.T) {
	candidate := Candidate{Selector: "crate::service::connect", Name: "connect", Language: "rust"}
	got, err := (Matcher{}).Retrieve(context.Background(), Query{Selector: "conenct", Language: "rust", Candidates: []Candidate{candidate}})
	if err != nil || len(got) != 1 || got[0].Selector != candidate.Selector {
		t.Fatalf("Rust leaf retrieval = %#v, %v", got, err)
	}
	got, err = (Matcher{}).Retrieve(context.Background(), Query{Selector: "crate::service::conenct", Language: "rust", Candidates: []Candidate{candidate}})
	if err != nil || len(got) != 1 || got[0].Selector != candidate.Selector {
		t.Fatalf("Rust qualified retrieval = %#v, %v", got, err)
	}
	got, err = (Matcher{}).Retrieve(context.Background(), Query{Selector: "Widgte", Language: "rust", Candidates: []Candidate{{Selector: "Widget", Name: "Widget", Language: "rust"}}})
	if err != nil || len(got) != 1 {
		t.Fatalf("Rust transposition retrieval = %#v, %v", got, err)
	}
}

func TestCodannaCLIArgumentsAndCandidates(t *testing.T) {
	bin := writeFakeCodanna(t, "printf '%s\\n' \"$@\" > \"$CAPTURE\"; printf '%s' '{\"results\":[{\"symbol\":{\"id\":7,\"name\":\"ReadConfig\",\"kind\":\"function\",\"file_path\":\"pkg/read.go\",\"language\":\"Go\"}}]}'")
	capture := filepath.Join(t.TempDir(), "args")
	t.Setenv("CAPTURE", capture)
	got, err := (Codanna{Binary: bin, Args: []string{"--profile", "test"}, Timeout: 15 * time.Second}).Retrieve(context.Background(), Query{Selector: "ReadConfig", Limit: 2})
	if err != nil || len(got) != 1 || got[0].Identity != "7" {
		t.Fatalf("Retrieve = %#v, %v", got, err)
	}
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--profile", "test", "mcp", "search_symbols", "query:ReadConfig", "limit:8", "--json"}
	if !reflect.DeepEqual(strings.Fields(string(data)), want) {
		t.Fatalf("args %q, want %#v", data, want)
	}
}

func TestCodannaCLIFailuresAreBounded(t *testing.T) {
	t.Run("absent", func(t *testing.T) {
		_, err := (Codanna{Binary: filepath.Join(t.TempDir(), "missing")}).Retrieve(context.Background(), Query{Selector: "Read"})
		if err == nil {
			t.Fatal("missing binary succeeded")
		}
	})
	t.Run("malformed", func(t *testing.T) {
		bin := writeFakeCodanna(t, "printf nope")
		_, err := (Codanna{Binary: bin, Timeout: 15 * time.Second}).Retrieve(context.Background(), Query{Selector: "Read"})
		if err == nil || !strings.Contains(err.Error(), "decode Codanna") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("oversized", func(t *testing.T) {
		bin := writeFakeCodanna(t, "printf '12345678901234567890'")
		_, err := (Codanna{Binary: bin, MaxOutputBytes: 8, Timeout: 15 * time.Second}).Retrieve(context.Background(), Query{Selector: "Read"})
		if !errors.Is(err, errOutputLimit) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("timeout", func(t *testing.T) {
		bin := writeFakeCodanna(t, "sleep 3")
		started := time.Now()
		_, err := (Codanna{Binary: bin, Timeout: 20 * time.Millisecond}).Retrieve(context.Background(), Query{Selector: "Read"})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error = %v", err)
		}
		if time.Since(started) > time.Second {
			t.Fatal("timeout was not bounded")
		}
	})
}

func TestCodannaWaitDelayBoundsInheritedPipe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX shell")
	}
	started := time.Now()
	_, err := (Codanna{Binary: "/bin/sh", Args: []string{"-c", "sleep 2 & wait"}, Timeout: 20 * time.Millisecond}).Retrieve(context.Background(), Query{Selector: "Read"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
	if time.Since(started) > 500*time.Millisecond {
		t.Fatalf("inherited pipe held command for %s", time.Since(started))
	}
}

func writeFakeCodanna(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "codanna")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

// WHY: Attributes benchmark gopls starts per module root so multi-module corpora do not look like process restarts.

package mcp_test

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/pipeline"
)

const (
	fileEditSessionRestart    = "RESTART"
	fileEditSessionNewSession = "NEW_SESSION"
	fileEditSessionLogLost    = "SESSION_LOG_LOST"
)

func fileEditSessionStarts(tb testing.TB, fixture managedMCPFixture) map[string]int {
	tb.Helper()
	data, err := os.ReadFile(fixture.counter + ".roots")
	if errors.Is(err, os.ErrNotExist) {
		return map[string]int{}
	}
	if err != nil {
		tb.Fatalf("read gopls session roots: %v", err)
	}
	starts := make(map[string]int)
	for root := range strings.SplitSeq(string(data), "\x00") {
		if root != "" {
			starts[root]++
		}
	}
	return starts
}

func fileEditSessionRestarted(starts map[string]int) bool {
	for _, count := range starts {
		if count > 1 {
			return true
		}
	}
	return false
}

func fileEditSessionChange(before, after map[string]int) string {
	for root, count := range after {
		prior := before[root]
		if count > prior && (prior > 0 || count > 1) {
			return fileEditSessionRestart
		}
	}
	for root, count := range before {
		if after[root] < count {
			return fileEditSessionLogLost
		}
	}
	for root, count := range after {
		if count > 0 && before[root] == 0 {
			return fileEditSessionNewSession
		}
	}
	return ""
}

func TestFileEditSessionStartsCountsRoots(t *testing.T) {
	fixture := managedMCPFixture{counter: t.TempDir() + "/starts"}
	if got := fileEditSessionStarts(t, fixture); len(got) != 0 {
		t.Fatalf("missing roots log = %v, want empty", got)
	}
	if err := pipeline.WriteAtomic(fixture.counter+".roots", []byte("/module/a\x00/module/b\x00/module/a\x00")); err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"/module/a": 2, "/module/b": 1}
	if got := fileEditSessionStarts(t, fixture); !reflect.DeepEqual(got, want) {
		t.Fatalf("session roots = %v, want %v", got, want)
	}
}

func TestFileEditSessionRestartedUsesPerRootCounts(t *testing.T) {
	tests := []struct {
		name   string
		starts map[string]int
		want   bool
	}{
		{name: "empty", starts: map[string]int{}, want: false},
		{name: "distinct module sessions", starts: map[string]int{"/a": 1, "/b": 1, "/c": 1}, want: false},
		{name: "same module restarted", starts: map[string]int{"/a": 2, "/b": 1}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := fileEditSessionRestarted(test.starts); got != test.want {
				t.Fatalf("fileEditSessionRestarted(%v) = %t, want %t", test.starts, got, test.want)
			}
		})
	}
}

func TestFileEditSessionChangeClassifiesRootIdentity(t *testing.T) {
	tests := []struct {
		name          string
		before, after map[string]int
		want          string
	}{
		{name: "unchanged", before: map[string]int{"/a": 1}, after: map[string]int{"/a": 1}},
		{name: "existing root restarted", before: map[string]int{"/a": 1}, after: map[string]int{"/a": 2}, want: fileEditSessionRestart},
		{name: "new module first session", before: map[string]int{"/a": 1}, after: map[string]int{"/a": 1, "/b": 1}, want: fileEditSessionNewSession},
		{name: "new module starts twice", before: map[string]int{"/a": 1}, after: map[string]int{"/a": 1, "/b": 2}, want: fileEditSessionRestart},
		{name: "restart outranks new module", before: map[string]int{"/a": 1}, after: map[string]int{"/a": 2, "/b": 1}, want: fileEditSessionRestart},
		{name: "empty log loses prior root", before: map[string]int{"/a": 1}, after: map[string]int{}, want: fileEditSessionLogLost},
		{name: "missing prior root", before: map[string]int{"/a": 1, "/b": 1}, after: map[string]int{"/a": 1}, want: fileEditSessionLogLost},
		{name: "decreased prior count", before: map[string]int{"/a": 2}, after: map[string]int{"/a": 1}, want: fileEditSessionLogLost},
		{name: "restart outranks lost root", before: map[string]int{"/a": 1, "/b": 1}, after: map[string]int{"/a": 2}, want: fileEditSessionRestart},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := fileEditSessionChange(test.before, test.after); got != test.want {
				t.Fatalf("fileEditSessionChange(%v, %v) = %q, want %q", test.before, test.after, got, test.want)
			}
		})
	}
}

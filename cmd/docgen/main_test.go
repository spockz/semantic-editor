// Tests protect the generated documentation contract so published examples remain usable without manual HTML fixes.
package main

import (
	"strings"
	"testing"
)

func TestRenderHTMLKeepsToolCallsContainedAndLinksExampleSources(t *testing.T) {
	t.Parallel()

	const filename = "insert_specialized.txtar"
	page := renderHTML(nil, nil, []TxtarExample{{Filename: filename}})

	wants := []string{
		"grid-template-columns: repeat(2, minmax(0, 1fr));",
		".inv-block {",
		"min-width: 0;",
		`href="https://github.com/spockz/semantic-editor/blob/main/testdata/scripts/` + filename + `"`,
	}
	for _, want := range wants {
		if !strings.Contains(page, want) {
			t.Errorf("rendered documentation does not contain %q", want)
		}
	}
}

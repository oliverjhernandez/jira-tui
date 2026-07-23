package main

import (
	"strings"
	"testing"
)

func TestProjectBoardJQL(t *testing.T) {
	jql := projectBoardJQL("TSIPC")
	for _, want := range []string{`project = "TSIPC"`, "issuetype in (Epic, Task)", "ORDER BY"} {
		if !strings.Contains(jql, want) {
			t.Errorf("projectBoardJQL = %q, missing %q", jql, want)
		}
	}
}

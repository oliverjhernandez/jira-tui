package main

import (
	"strings"
	"testing"
)

func TestHelpGroupsWellFormed(t *testing.T) {
	if len(helpGroups) == 0 {
		t.Fatal("helpGroups is empty")
	}
	for _, g := range helpGroups {
		if strings.TrimSpace(g.title) == "" {
			t.Errorf("group with empty title: %+v", g)
		}
		if len(g.binds) == 0 {
			t.Errorf("group %q has no binds", g.title)
		}
		for _, b := range g.binds {
			if strings.TrimSpace(b.keys) == "" || strings.TrimSpace(b.desc) == "" {
				t.Errorf("group %q has a blank bind: %+v", g.title, b)
			}
		}
	}
}

func TestBuildHelpContentMentionsKeys(t *testing.T) {
	content := buildHelpContent()
	for _, want := range []string{"gt", "?", "Set estimate", "Half page"} {
		if !strings.Contains(content, want) {
			t.Errorf("help content missing %q", want)
		}
	}
}

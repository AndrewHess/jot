package jot

import (
	"os"
	"testing"
)

func TestUpdateCheckboxLine(t *testing.T) {
	line, err := updateCheckboxLine("- [ ] write tests", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "- [x] write tests" {
		t.Fatalf("unexpected updated line: %q", line)
	}

	line, err = updateCheckboxLine("- [x] write tests", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "- [ ] write tests" {
		t.Fatalf("unexpected updated line: %q", line)
	}
}

func TestUpdateCheckboxLineRejectsNonCheckbox(t *testing.T) {
	_, err := updateCheckboxLine("- write tests", true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeTopicName(t *testing.T) {
	got := normalizeTopicName("feature/auth-flow")
	if got != "feature-auth-flow" {
		t.Fatalf("unexpected normalized topic %q", got)
	}
}

func TestResolveTopicExplicit(t *testing.T) {
	topic, source, err := resolveTopic("foo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topic != "foo" || source != "explicit" {
		t.Fatalf("unexpected result: topic=%q source=%q", topic, source)
	}
}

func TestResolveTopicRequiresExplicitOutsideGit(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}

	_, _, err = resolveTopic("", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveTopicForced(t *testing.T) {
	topic, source, err := resolveTopic("", "later")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topic != "later" || source != "explicit" {
		t.Fatalf("unexpected result: topic=%q source=%q", topic, source)
	}
}

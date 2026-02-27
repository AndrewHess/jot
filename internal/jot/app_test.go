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

func TestLaterTopicNameDefaultAndEnv(t *testing.T) {
	t.Setenv(laterTopicEnv, "")
	if got := laterTopicName(); got != "later" {
		t.Fatalf("expected later, got %q", got)
	}

	t.Setenv(laterTopicEnv, "inbox/next-up")
	if got := laterTopicName(); got != "inbox-next-up" {
		t.Fatalf("unexpected env topic %q", got)
	}

	_ = os.Unsetenv(laterTopicEnv)
}

func TestChooseActiveTopic(t *testing.T) {
	topic, source, err := chooseActiveTopic("main", "feature-auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topic != "feature-auth" || source != "git branch" {
		t.Fatalf("unexpected result: topic=%q source=%q", topic, source)
	}

	topic, source, err = chooseActiveTopic("foo", "feature-auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topic != "foo" || source != "jot state" {
		t.Fatalf("unexpected result: topic=%q source=%q", topic, source)
	}
}

package jot

import (
	"bytes"
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
	if got != "feature/auth-flow" {
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

func TestResolveTopicRejectsDotTopics(t *testing.T) {
	_, _, err := resolveTopic(".", "")
	if err == nil {
		t.Fatal("expected error for dot topic")
	}

	_, _, err = resolveTopic("..", "")
	if err == nil {
		t.Fatal("expected error for dot-dot topic")
	}
}

func TestResolveTopicAllowsSlashTopic(t *testing.T) {
	topic, source, err := resolveTopic("foo/bar", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topic != "foo/bar" || source != "explicit" {
		t.Fatalf("unexpected result: topic=%q source=%q", topic, source)
	}
}

func TestResolveTopicRejectsUnsafeSlashTopics(t *testing.T) {
	invalid := []string{"/foo", "foo//bar", "foo/../bar", "foo/./bar"}
	for _, topic := range invalid {
		_, _, err := resolveTopic(topic, "")
		if err == nil {
			t.Fatalf("expected error for topic %q", topic)
		}
	}
}

func TestAddLinesFromStdin(t *testing.T) {
	app, err := NewApp(bytes.NewBufferString("first\n\nsecond\n"), &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("NewApp returned error: %v", err)
	}

	lines, err := app.addLines(AddOptions{Checkbox: false})
	if err != nil {
		t.Fatalf("addLines returned error: %v", err)
	}
	if len(lines) != 2 || lines[0] != "- first" || lines[1] != "- second" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}

func TestNumberLinesWidth(t *testing.T) {
	lines := numberLines([]string{"first", "second"})
	if len(lines) != 2 {
		t.Fatalf("unexpected line count: %d", len(lines))
	}
	if lines[0] != " 1 | first" || lines[1] != " 2 | second" {
		t.Fatalf("unexpected numbered lines: %#v", lines)
	}
}

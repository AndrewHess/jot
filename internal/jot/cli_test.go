package jot

import "testing"

func TestParseAddWithCheckbox(t *testing.T) {
	cmd, err := Parse([]string{"add", "-c", "ship", "it"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != CommandAdd {
		t.Fatalf("expected CommandAdd, got %v", cmd.Kind)
	}
	if !cmd.AddOptions.Checkbox {
		t.Fatalf("expected checkbox=true")
	}
	if cmd.AddOptions.Text != "ship it" {
		t.Fatalf("unexpected text %q", cmd.AddOptions.Text)
	}
}

func TestParseAddWithTopic(t *testing.T) {
	cmd, err := Parse([]string{"add", "-t", "later/queue", "capture", "this"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != CommandAdd {
		t.Fatalf("expected CommandAdd, got %v", cmd.Kind)
	}
	if cmd.AddOptions.Topic != "later/queue" {
		t.Fatalf("unexpected topic %q", cmd.AddOptions.Topic)
	}
}

func TestParseAddWithoutText(t *testing.T) {
	cmd, err := Parse([]string{"add", "-c"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != CommandAdd {
		t.Fatalf("expected CommandAdd, got %v", cmd.Kind)
	}
	if !cmd.AddOptions.Checkbox {
		t.Fatalf("expected checkbox=true")
	}
	if cmd.AddOptions.Text != "" {
		t.Fatalf("expected empty text, got %q", cmd.AddOptions.Text)
	}
}

func TestParseShowWithTopic(t *testing.T) {
	cmd, err := Parse([]string{"show", "-t", "later"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != CommandShow {
		t.Fatalf("expected CommandShow, got %v", cmd.Kind)
	}
	if cmd.Topic != "later" {
		t.Fatalf("unexpected topic %q", cmd.Topic)
	}
}

func TestParseUnknownCommand(t *testing.T) {
	_, err := Parse([]string{"wat"})
	if err == nil {
		t.Fatal("expected error")
	}
}

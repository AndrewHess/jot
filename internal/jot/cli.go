package jot

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

type Command int

const (
	CommandHelp Command = iota
	CommandInit
	CommandAdd
	CommandShow
	CommandEdit
	CommandDone
	CommandUndone
	CommandStatus
)

type UsageError struct {
	Message string
}

func (e *UsageError) Error() string {
	return e.Message
}

type AddOptions struct {
	Checkbox bool
	Topic    string
	Text     string
}

type ParsedCommand struct {
	Kind         Command
	Topic        string
	LineNumber   int
	AddOptions   AddOptions
	OriginalArgs []string
}

func Run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	parsed, err := Parse(args)
	if err != nil {
		return err
	}

	app, err := NewApp(stdin, stdout, stderr)
	if err != nil {
		return err
	}

	switch parsed.Kind {
	case CommandHelp:
		app.PrintUsage()
		return nil
	case CommandInit:
		return app.Init()
	case CommandAdd:
		return app.Add(parsed.AddOptions)
	case CommandShow:
		return app.Show(parsed.Topic)
	case CommandEdit:
		return app.Edit(parsed.Topic)
	case CommandDone:
		return app.SetCheckbox(parsed.LineNumber, true, parsed.Topic)
	case CommandUndone:
		return app.SetCheckbox(parsed.LineNumber, false, parsed.Topic)
	case CommandStatus:
		return app.Status(parsed.Topic)
	}

	return fmt.Errorf("unhandled command: %v", parsed.Kind)
}

func Parse(args []string) (ParsedCommand, error) {
	if len(args) == 0 {
		return ParsedCommand{Kind: CommandHelp, OriginalArgs: args}, nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		return ParsedCommand{Kind: CommandHelp, OriginalArgs: args}, nil
	case "init":
		return ParsedCommand{Kind: CommandInit, OriginalArgs: args}, nil
	case "add":
		return parseAdd(args)
	case "show", "cat":
		topic, err := parseTopicFlag(args[1:], "usage: jot show [-t|--topic <topic>]")
		if err != nil {
			return ParsedCommand{}, err
		}
		return ParsedCommand{Kind: CommandShow, Topic: topic, OriginalArgs: args}, nil
	case "edit":
		topic, err := parseTopicFlag(args[1:], "usage: jot edit [-t|--topic <topic>]")
		if err != nil {
			return ParsedCommand{}, err
		}
		return ParsedCommand{Kind: CommandEdit, Topic: topic, OriginalArgs: args}, nil
	case "done":
		line, topic, err := parseLineNumberAndTopic(args[1:], "usage: jot done [-t|--topic <topic>] <line-number>")
		if err != nil {
			return ParsedCommand{}, err
		}
		return ParsedCommand{Kind: CommandDone, LineNumber: line, Topic: topic, OriginalArgs: args}, nil
	case "undone":
		line, topic, err := parseLineNumberAndTopic(args[1:], "usage: jot undone [-t|--topic <topic>] <line-number>")
		if err != nil {
			return ParsedCommand{}, err
		}
		return ParsedCommand{Kind: CommandUndone, LineNumber: line, Topic: topic, OriginalArgs: args}, nil
	case "status":
		topic, err := parseTopicFlag(args[1:], "usage: jot status [-t|--topic <topic>]")
		if err != nil {
			return ParsedCommand{}, err
		}
		return ParsedCommand{Kind: CommandStatus, Topic: topic, OriginalArgs: args}, nil
	default:
		return ParsedCommand{}, &UsageError{Message: fmt.Sprintf("unknown command: %s\n\n%s", args[0], usageText)}
	}
}

func parseAdd(args []string) (ParsedCommand, error) {
	opts, err := parseAddOptions(args[1:], "usage: jot add [-c|--checkbox] [-t|--topic <topic>] <text>")
	if err != nil {
		return ParsedCommand{}, err
	}

	return ParsedCommand{
		Kind:         CommandAdd,
		AddOptions:   opts,
		OriginalArgs: args,
	}, nil
}

func parseAddOptions(rawArgs []string, usage string) (AddOptions, error) {
	addFlags := flag.NewFlagSet("add", flag.ContinueOnError)
	addFlags.SetOutput(io.Discard)

	var checkbox bool
	var topic string
	addFlags.BoolVar(&checkbox, "checkbox", false, "add as checkbox")
	addFlags.BoolVar(&checkbox, "c", false, "add as checkbox")
	addFlags.StringVar(&topic, "topic", "", "write to topic without switching")
	addFlags.StringVar(&topic, "t", "", "write to topic without switching")

	if err := addFlags.Parse(rawArgs); err != nil {
		return AddOptions{}, &UsageError{Message: usage}
	}

	text := strings.TrimSpace(strings.Join(addFlags.Args(), " "))
	if text == "" {
		return AddOptions{}, &UsageError{Message: usage}
	}

	return AddOptions{
		Checkbox: checkbox,
		Topic:    strings.TrimSpace(topic),
		Text:     text,
	}, nil
}

func parseTopicFlag(rawArgs []string, usage string) (string, error) {
	fs := flag.NewFlagSet("topic", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var topic string
	fs.StringVar(&topic, "topic", "", "topic override")
	fs.StringVar(&topic, "t", "", "topic override")

	if err := fs.Parse(rawArgs); err != nil {
		return "", &UsageError{Message: usage}
	}
	if len(fs.Args()) != 0 {
		return "", &UsageError{Message: usage}
	}
	return strings.TrimSpace(topic), nil
}

func parseLineNumberAndTopic(rawArgs []string, usage string) (int, string, error) {
	fs := flag.NewFlagSet("line", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var topic string
	fs.StringVar(&topic, "topic", "", "topic override")
	fs.StringVar(&topic, "t", "", "topic override")
	if err := fs.Parse(rawArgs); err != nil {
		return 0, "", &UsageError{Message: usage}
	}

	if len(fs.Args()) != 1 {
		return 0, "", &UsageError{Message: usage}
	}

	n, err := parsePositiveInt(fs.Args()[0])
	if err != nil {
		return 0, "", &UsageError{Message: usage}
	}
	return n, strings.TrimSpace(topic), nil
}

func parsePositiveInt(raw string) (int, error) {
	var n int
	_, err := fmt.Sscanf(raw, "%d", &n)
	if err != nil || n < 1 {
		return 0, errors.New("invalid integer")
	}
	return n, nil
}

const usageText = `jot - scratchpad CLI for topic-based notes

Usage:
  jot <command> [arguments]

Commands:
  init
      Initialize .jot in the current project (or nearest parent root).
  add [-c|--checkbox] [-t|--topic <topic>] <text>
      Append a note. Use -c for a markdown checkbox item.
  show [-t|--topic <topic>]
      Print the active topic file.
      Alias: cat
  edit [-t|--topic <topic>]
      Open the active topic file in $VISUAL, then $EDITOR, then nvim, then vi.
  done [-t|--topic <topic>] <line-number>
      Mark checkbox at line as complete (- [x] ...).
  undone [-t|--topic <topic>] <line-number>
      Mark checkbox at line as incomplete (- [ ] ...).
  status [-t|--topic <topic>]
      Show active root, topic source, and topic file path.
  help
      Show this message.
`

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
	CommandUse
	CommandAdd
	CommandLater
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
	case CommandUse:
		return app.UseTopic(parsed.Topic)
	case CommandAdd:
		return app.Add(parsed.AddOptions)
	case CommandLater:
		return app.AddToLater(parsed.AddOptions)
	case CommandShow:
		return app.Show()
	case CommandEdit:
		return app.Edit()
	case CommandDone:
		return app.SetCheckbox(parsed.LineNumber, true)
	case CommandUndone:
		return app.SetCheckbox(parsed.LineNumber, false)
	case CommandStatus:
		return app.Status()
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
	case "use", "checkout":
		if len(args) != 2 {
			return ParsedCommand{}, &UsageError{Message: "usage: jot use <topic>"}
		}
		return ParsedCommand{Kind: CommandUse, Topic: args[1], OriginalArgs: args}, nil
	case "add":
		return parseAdd(args)
	case "later":
		return parseLater(args)
	case "show", "cat":
		return ParsedCommand{Kind: CommandShow, OriginalArgs: args}, nil
	case "edit":
		return ParsedCommand{Kind: CommandEdit, OriginalArgs: args}, nil
	case "done":
		line, err := parseLineNumber(args, "usage: jot done <line-number>")
		if err != nil {
			return ParsedCommand{}, err
		}
		return ParsedCommand{Kind: CommandDone, LineNumber: line, OriginalArgs: args}, nil
	case "undone":
		line, err := parseLineNumber(args, "usage: jot undone <line-number>")
		if err != nil {
			return ParsedCommand{}, err
		}
		return ParsedCommand{Kind: CommandUndone, LineNumber: line, OriginalArgs: args}, nil
	case "status":
		return ParsedCommand{Kind: CommandStatus, OriginalArgs: args}, nil
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

func parseLater(args []string) (ParsedCommand, error) {
	opts, err := parseAddOptions(args[1:], "usage: jot later [-c|--checkbox] <text>")
	if err != nil {
		return ParsedCommand{}, err
	}
	if opts.Topic != "" {
		return ParsedCommand{}, &UsageError{Message: "usage: jot later [-c|--checkbox] <text>"}
	}

	return ParsedCommand{
		Kind:         CommandLater,
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

func parseLineNumber(args []string, usage string) (int, error) {
	if len(args) != 2 {
		return 0, &UsageError{Message: usage}
	}
	n, err := parsePositiveInt(args[1])
	if err != nil {
		return 0, &UsageError{Message: usage}
	}
	return n, nil
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
  use <topic>
      Set the persisted topic in .jot/state.json.
      Alias: checkout
  add [-c|--checkbox] [-t|--topic <topic>] <text>
      Append a note. Use -c for a markdown checkbox item.
      Use -t to add to another topic without switching state.
  later [-c|--checkbox] <text>
      Add to the "later" topic (or $JOT_LATER_TOPIC if set).
  show
      Print the active topic file.
      Alias: cat
  edit
      Open the active topic file in $VISUAL, then $EDITOR, then nvim, then vi.
  done <line-number>
      Mark checkbox at line as complete (- [x] ...).
  undone <line-number>
      Mark checkbox at line as incomplete (- [ ] ...).
  status
      Show active root, topic source, and topic file path.
  help
      Show this message.

Topic resolution:
  1. If inside a git worktree with a current branch, branch name is used as topic.
  2. Else the persisted .jot state topic is used.
  3. add -t / --topic and later override both for that single command.

Storage:
  - .jot/state.json
  - .jot/topics/<topic>.md

Examples:
  jot add "capture a quick note"
  jot add -c "follow up with QA"
  jot add -t later "not part of this story"
  jot later -c "revisit after release"
  jot show
  jot done 3
`

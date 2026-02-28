# jot

`jot` is a lightweight CLI scratchpad with topic-based context, similar to branch-style workflows.

## User Guide

Keep notes while coding without breaking flow:
- Use your current git branch as the default topic
- Persist notes in `.jot/topics`
- Add quick bullets or checkbox items

### Command reference

Use:

```bash
jot --help
```

### Behavior

- In a git repository, the active topic defaults to your current branch name.
- Outside git, pass `-t <topic>` for topic-dependent commands.
- `jot add -t <topic> ...` writes to any topic as a one-off without switching context.
- `jot add` with no `[text]` reads from stdin until EOF (`Ctrl-D` on a blank line).

### Storage layout

```text
.jot/
  topics/
    <topic>.md
```

### macOS note

- macOS ships a system `jot` at `/usr/bin/jot`.
- In development, run `./bin/jot` or `just run ...` to ensure you are using this project.
- After Homebrew install, make sure Homebrew's bin path is before `/usr/bin` in your `PATH`.

## Development

```bash
just build
./bin/jot --help
```

### Common commands

```bash
just fmt
just test
just lint
just run --help
```

### Linting

`golangci-lint` is configured in `.golangci.yml`, including `exhaustive` checks for enum switches.

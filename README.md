# jot

`jot` is a lightweight CLI scratchpad with topic-based context, similar to branch-style workflows.

## Why

Keep notes while coding without breaking flow:
- Persist state in `.jot/state.json`
- Keep one active topic at a time
- Add quick bullets or checkbox items

## Install (local dev)

```bash
just build
./bin/jot --help
```

Note for macOS:
- macOS ships a system `jot` at `/usr/bin/jot`.
- In development, run `./bin/jot` or `just run ...` to ensure you are using this project.
- After Homebrew install, make sure Homebrew's bin path is before `/usr/bin` in your `PATH`.

## Dev commands

```bash
just fmt
just test
just lint
just run --help
```

## Commands

```text
jot init
jot add [-c|--checkbox] [-t|--topic <topic>] <text>
jot later [-c|--checkbox] <text>
jot show [-t|--topic <topic>]
jot edit [-t|--topic <topic>]
jot done [-t|--topic <topic>] <line-number>
jot undone [-t|--topic <topic>] <line-number>
jot status [-t|--topic <topic>]
jot help | jot -h | jot --help
```

Behavior notes:
- In a git repository, the active topic defaults to your current branch name.
- Outside git, pass `-t <topic>` for topic-dependent commands.
- `jot add -t <topic> ...` writes to another topic without switching current context.
- `jot later ...` is shorthand for adding to topic `later` (override with `JOT_LATER_TOPIC`).

## Layout

```text
.jot/
  state.json
  topics/
    <topic>.md
```

## Examples

```bash
jot init
jot add "triage flaky test in CI"
jot add -c "submit fix PR"
jot add "repro only on arm64"
jot add -t later "follow up on flaky benchmark"
jot later -c "revisit this after release"
jot show
jot done 1
jot edit
# outside git:
jot show -t later
```

## Linting

`golangci-lint` is configured in `.golangci.yml`, including `exhaustive` checks for enum switches.

## Homebrew release plan

This repo includes `.goreleaser.yaml` with a `brew` stanza for publishing to a tap repo:
- Target tap repo: `andrewhess/homebrew-tap`
- Formula generated on release tags

Before first release:
- Create `andrewhess/homebrew-tap`
- Update `commit_author` in `.goreleaser.yaml`
- Tag a release (example: `v0.1.0`)

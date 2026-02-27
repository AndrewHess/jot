package jot

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	toolDirName   = ".jot"
	topicsDirName = "topics"
	stateFileName = "state.json"
	defaultTopic  = "main"
	laterTopicEnv = "JOT_LATER_TOPIC"
)

var topicPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type State struct {
	Version      int    `json:"version"`
	CurrentTopic string `json:"current_topic"`
}

type App struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func NewApp(stdin io.Reader, stdout io.Writer, stderr io.Writer) (*App, error) {
	return &App{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}, nil
}

func (a *App) PrintUsage() {
	fmt.Fprint(a.stdout, usageText)
}

func (a *App) Init() error {
	root, found, err := findRootFromWD()
	if err != nil {
		return err
	}
	if !found {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return wdErr
		}
		root = wd
	}

	_, err = ensureInitialized(root)
	if err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "initialized jot in %s\n", filepath.Join(root, toolDirName))
	return nil
}

func (a *App) UseTopic(topic string) error {
	if !isTopicNameValid(topic) {
		return fmt.Errorf("invalid topic %q: only [A-Za-z0-9._-] are allowed", topic)
	}

	root, err := ensureWorkingRoot()
	if err != nil {
		return err
	}

	state, paths, err := loadState(root)
	if err != nil {
		return err
	}
	state.CurrentTopic = topic
	if err := saveState(paths.StatePath, state); err != nil {
		return err
	}
	if err := ensureTopicFile(paths.TopicsDir, topic); err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "current topic: %s\n", topic)
	return nil
}

func (a *App) Add(options AddOptions) error {
	return a.addToTopic(options, "")
}

func (a *App) AddToLater(options AddOptions) error {
	return a.addToTopic(options, laterTopicName())
}

func (a *App) addToTopic(options AddOptions, forcedTopic string) error {
	root, err := ensureWorkingRoot()
	if err != nil {
		return err
	}

	state, paths, err := loadState(root)
	if err != nil {
		return err
	}

	targetTopic, err := resolveTopic(state, options.Topic, forcedTopic)
	if err != nil {
		return err
	}
	if err := ensureTopicFile(paths.TopicsDir, targetTopic); err != nil {
		return err
	}

	topicPath := filepath.Join(paths.TopicsDir, targetTopic+".md")
	line := "- " + options.Text
	if options.Checkbox {
		line = "- [ ] " + options.Text
	}

	f, err := os.OpenFile(topicPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := fmt.Fprintln(f, line); err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "added to %s\n", targetTopic)
	return nil
}

func (a *App) Show() error {
	root, err := ensureWorkingRoot()
	if err != nil {
		return err
	}

	state, paths, err := loadState(root)
	if err != nil {
		return err
	}

	topic, err := resolveActiveTopic(state)
	if err != nil {
		return err
	}

	topicPath := filepath.Join(paths.TopicsDir, topic+".md")
	if err := ensureTopicFile(paths.TopicsDir, topic); err != nil {
		return err
	}
	content, err := os.ReadFile(topicPath)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		fmt.Fprintln(a.stdout, "(empty)")
		return nil
	}
	_, err = a.stdout.Write(content)
	return err
}

func (a *App) Edit() error {
	root, err := ensureWorkingRoot()
	if err != nil {
		return err
	}

	state, paths, err := loadState(root)
	if err != nil {
		return err
	}

	topic, err := resolveActiveTopic(state)
	if err != nil {
		return err
	}

	topicPath := filepath.Join(paths.TopicsDir, topic+".md")
	if err := ensureTopicFile(paths.TopicsDir, topic); err != nil {
		return err
	}
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "nvim"
	}

	cmd := exec.Command(editor, topicPath)
	cmd.Stdin = a.stdin
	cmd.Stdout = a.stdout
	cmd.Stderr = a.stderr
	err = cmd.Run()
	if err == nil {
		return nil
	}

	if errors.Is(err, exec.ErrNotFound) && editor == "nvim" {
		cmd = exec.Command("vi", topicPath)
		cmd.Stdin = a.stdin
		cmd.Stdout = a.stdout
		cmd.Stderr = a.stderr
		return cmd.Run()
	}

	return fmt.Errorf("failed to launch editor %q: %w", editor, err)
}

func (a *App) SetCheckbox(lineNumber int, done bool) error {
	root, err := ensureWorkingRoot()
	if err != nil {
		return err
	}

	state, paths, err := loadState(root)
	if err != nil {
		return err
	}
	topic, err := resolveActiveTopic(state)
	if err != nil {
		return err
	}

	topicPath := filepath.Join(paths.TopicsDir, topic+".md")
	if err := ensureTopicFile(paths.TopicsDir, topic); err != nil {
		return err
	}

	content, err := os.ReadFile(topicPath)
	if err != nil {
		return err
	}
	lines := splitLines(string(content))
	if lineNumber > len(lines) {
		return fmt.Errorf("line %d does not exist", lineNumber)
	}

	idx := lineNumber - 1
	updated, err := updateCheckboxLine(lines[idx], done)
	if err != nil {
		return err
	}
	lines[idx] = updated

	result := strings.Join(lines, "\n")
	if strings.TrimSpace(result) != "" && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	return os.WriteFile(topicPath, []byte(result), 0o644)
}

func (a *App) Status() error {
	root, err := ensureWorkingRoot()
	if err != nil {
		return err
	}

	state, paths, err := loadState(root)
	if err != nil {
		return err
	}

	topic, source, err := resolveActiveTopicWithSource(state)
	if err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "root: %s\n", root)
	fmt.Fprintf(a.stdout, "topic: %s\n", topic)
	fmt.Fprintf(a.stdout, "source: %s\n", source)
	fmt.Fprintf(a.stdout, "file: %s\n", filepath.Join(paths.TopicsDir, topic+".md"))
	return nil
}

func splitLines(content string) []string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	lines := make([]string, 0, 16)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func updateCheckboxLine(line string, done bool) (string, error) {
	const (
		open  = "- [ ] "
		close = "- [x] "
	)

	switch {
	case strings.HasPrefix(line, open):
		if done {
			return close + strings.TrimPrefix(line, open), nil
		}
		return line, nil
	case strings.HasPrefix(line, close):
		if done {
			return line, nil
		}
		return open + strings.TrimPrefix(line, close), nil
	default:
		return "", errors.New("target line is not a checkbox item")
	}
}

func isTopicNameValid(topic string) bool {
	return topicPattern.MatchString(topic)
}

func resolveTopic(state State, explicitTopic string, forcedTopic string) (string, error) {
	switch {
	case forcedTopic != "":
		if !isTopicNameValid(forcedTopic) {
			return "", fmt.Errorf("invalid topic %q: only [A-Za-z0-9._-] are allowed", forcedTopic)
		}
		return forcedTopic, nil
	case explicitTopic != "":
		if !isTopicNameValid(explicitTopic) {
			return "", fmt.Errorf("invalid topic %q: only [A-Za-z0-9._-] are allowed", explicitTopic)
		}
		return explicitTopic, nil
	default:
		return resolveActiveTopic(state)
	}
}

func resolveActiveTopic(state State) (string, error) {
	topic, _, err := resolveActiveTopicWithSource(state)
	return topic, err
}

func resolveActiveTopicWithSource(state State) (string, string, error) {
	gitTopic, _ := gitBranchTopic()
	return chooseActiveTopic(state.CurrentTopic, gitTopic)
}

func chooseActiveTopic(stateTopic string, gitTopic string) (string, string, error) {
	if strings.TrimSpace(stateTopic) == "" {
		stateTopic = defaultTopic
	}
	if !isTopicNameValid(stateTopic) {
		return "", "", fmt.Errorf("invalid current topic in state: %q", stateTopic)
	}

	if gitTopic != "" && stateTopic == defaultTopic {
		return gitTopic, "git branch", nil
	}
	return stateTopic, "jot state", nil
}

func gitBranchTopic() (string, bool) {
	cmd := exec.Command("git", "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}

	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "", false
	}
	topic := normalizeTopicName(branch)
	if topic == "" || !isTopicNameValid(topic) {
		return "", false
	}
	return topic, true
}

func normalizeTopicName(raw string) string {
	raw = strings.TrimSpace(raw)
	var b strings.Builder
	b.Grow(len(raw))

	prevDash := false
	for _, r := range raw {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}

	result := strings.Trim(b.String(), "-")
	if result == "" {
		return ""
	}
	return result
}

func laterTopicName() string {
	topic := strings.TrimSpace(os.Getenv(laterTopicEnv))
	if topic == "" {
		return "later"
	}
	return normalizeTopicName(topic)
}

type Paths struct {
	JotDir    string
	TopicsDir string
	StatePath string
}

func ensureWorkingRoot() (string, error) {
	root, found, err := findRootFromWD()
	if err != nil {
		return "", err
	}
	if found {
		return root, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	_, err = ensureInitialized(wd)
	return wd, err
}

func ensureInitialized(root string) (Paths, error) {
	jotDir := filepath.Join(root, toolDirName)
	topicsDir := filepath.Join(jotDir, topicsDirName)
	statePath := filepath.Join(jotDir, stateFileName)

	if err := os.MkdirAll(topicsDir, 0o755); err != nil {
		return Paths{}, err
	}

	if _, err := os.Stat(statePath); errors.Is(err, os.ErrNotExist) {
		initial := State{Version: 1, CurrentTopic: defaultTopic}
		if err := saveState(statePath, initial); err != nil {
			return Paths{}, err
		}
		if err := ensureTopicFile(topicsDir, defaultTopic); err != nil {
			return Paths{}, err
		}
	} else if err != nil {
		return Paths{}, err
	}

	return Paths{
		JotDir:    jotDir,
		TopicsDir: topicsDir,
		StatePath: statePath,
	}, nil
}

func ensureTopicFile(topicsDir string, topic string) error {
	path := filepath.Join(topicsDir, topic+".md")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(path, []byte(""), 0o644)
	} else {
		return err
	}
}

func loadState(root string) (State, Paths, error) {
	paths, err := ensureInitialized(root)
	if err != nil {
		return State{}, Paths{}, err
	}

	raw, err := os.ReadFile(paths.StatePath)
	if err != nil {
		return State{}, Paths{}, err
	}

	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return State{}, Paths{}, err
	}
	if state.CurrentTopic == "" {
		state.CurrentTopic = defaultTopic
	}
	if !isTopicNameValid(state.CurrentTopic) {
		return State{}, Paths{}, fmt.Errorf("invalid current topic in state: %q", state.CurrentTopic)
	}
	if err := ensureTopicFile(paths.TopicsDir, state.CurrentTopic); err != nil {
		return State{}, Paths{}, err
	}
	return state, paths, nil
}

func saveState(path string, state State) error {
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func findRootFromWD() (string, bool, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false, err
	}

	for {
		jotPath := filepath.Join(dir, toolDirName)
		info, err := os.Stat(jotPath)
		if err == nil && info.IsDir() {
			return dir, true, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", false, err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, nil
		}
		dir = parent
	}
}

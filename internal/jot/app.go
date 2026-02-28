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
	"strconv"
	"strings"
)

const (
	toolDirName   = ".jot"
	topicsDirName = "topics"
	stateFileName = "state.json"
	colorStart    = "\033[38;5;214m"
	colorEnd      = "\033[0m"
)

var topicSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type State struct {
	Version int `json:"version"`
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
	_, _ = fmt.Fprint(a.stdout, usageText)
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

	if _, err := fmt.Fprintf(a.stdout, "initialized jot in %s\n", filepath.Join(root, toolDirName)); err != nil {
		return err
	}
	return nil
}

func (a *App) Add(options AddOptions) error {
	return a.addToTopic(options, "")
}

func (a *App) addToTopic(options AddOptions, forcedTopic string) error {
	root, err := ensureWorkingRoot()
	if err != nil {
		return err
	}

	paths, err := ensurePaths(root)
	if err != nil {
		return err
	}

	targetTopic, _, err := resolveTopic(options.Topic, forcedTopic)
	if err != nil {
		return err
	}
	if err := ensureTopicFile(paths.TopicsDir, targetTopic); err != nil {
		return err
	}

	lines, err := a.addLines(options)
	if err != nil {
		return err
	}

	topicPath := filepath.Join(paths.TopicsDir, targetTopic+".md")

	f, err := os.OpenFile(topicPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	for _, line := range lines {
		if _, err := fmt.Fprintln(f, line); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(a.stdout, a.metaLine(fmt.Sprintf("added to topic %s", targetTopic))); err != nil {
		return err
	}
	return nil
}

func (a *App) addLines(options AddOptions) ([]string, error) {
	text := strings.TrimSpace(options.Text)
	if text == "" {
		if isTTYReader(a.stdin) {
			if _, err := fmt.Fprintln(a.stdout, a.metaLine("Enter note text. Press Ctrl-D on a blank line to finish.")); err != nil {
				return nil, err
			}
		}
		raw, err := io.ReadAll(a.stdin)
		if err != nil {
			return nil, err
		}
		text = strings.TrimSpace(string(raw))
	}
	if text == "" {
		return nil, &UsageError{Message: "usage: jot add [-c|--checkbox] [-t|--topic <topic>] [text]"}
	}

	lines := splitNonEmptyLines(text)
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if options.Checkbox {
			result = append(result, "- [ ] "+line)
			continue
		}
		result = append(result, "- "+line)
	}
	return result, nil
}

func (a *App) Show(topicOverride string) error {
	root, err := ensureWorkingRoot()
	if err != nil {
		return err
	}

	paths, err := ensurePaths(root)
	if err != nil {
		return err
	}

	topic, _, err := resolveTopic(topicOverride, "")
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
		if _, err := fmt.Fprintln(a.stdout, a.tagLine("empty")); err != nil {
			return err
		}
		return nil
	}

	lines := splitLines(string(content))
	for _, line := range numberLines(lines, isTTYWriter(a.stdout)) {
		if _, err := fmt.Fprintln(a.stdout, line); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) Edit(topicOverride string) error {
	root, err := ensureWorkingRoot()
	if err != nil {
		return err
	}

	paths, err := ensurePaths(root)
	if err != nil {
		return err
	}

	topic, _, err := resolveTopic(topicOverride, "")
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

func (a *App) SetCheckbox(lineNumber int, done bool, topicOverride string) error {
	root, err := ensureWorkingRoot()
	if err != nil {
		return err
	}

	paths, err := ensurePaths(root)
	if err != nil {
		return err
	}
	topic, _, err := resolveTopic(topicOverride, "")
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

func (a *App) Status(topicOverride string) error {
	root, err := ensureWorkingRoot()
	if err != nil {
		return err
	}

	paths, err := ensurePaths(root)
	if err != nil {
		return err
	}

	topic, source, err := resolveTopic(topicOverride, "")
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(a.stdout, "root: %s\n", root); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "topic: %s\n", topic); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "source: %s\n", source); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "file: %s\n", filepath.Join(paths.TopicsDir, topic+".md")); err != nil {
		return err
	}
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

func splitNonEmptyLines(content string) []string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	lines := make([]string, 0, 8)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
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
	if topic == "" {
		return false
	}
	if strings.HasPrefix(topic, "/") || strings.Contains(topic, "\\") {
		return false
	}

	parts := strings.Split(topic, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
		if !topicSegmentPattern.MatchString(part) {
			return false
		}
	}

	return true
}

func resolveTopic(explicitTopic string, forcedTopic string) (string, string, error) {
	switch {
	case forcedTopic != "":
		if !isTopicNameValid(forcedTopic) {
			return "", "", invalidTopicError(forcedTopic)
		}
		return forcedTopic, "explicit", nil
	case explicitTopic != "":
		if !isTopicNameValid(explicitTopic) {
			return "", "", invalidTopicError(explicitTopic)
		}
		return explicitTopic, "explicit", nil
	default:
		if topic, ok := gitBranchTopic(); ok {
			return topic, "git branch", nil
		}
		return "", "", errors.New("unable to resolve topic outside a git branch; pass -t <topic>")
	}
}

func invalidTopicError(topic string) error {
	return fmt.Errorf("invalid topic %q: must match ^[A-Za-z0-9._-]+(/[A-Za-z0-9._-]+)*$ (segments cannot be . or ..)", topic)
}

func (a *App) metaLine(message string) string {
	return a.tagLine("jot") + " " + message
}

func (a *App) tagLine(tag string) string {
	segment := "[" + tag + "]"
	// Use styling only for terminal output so redirected output stays clean.
	if isTTYWriter(a.stdout) {
		return colorStart + segment + colorEnd
	}
	return segment
}

func isTTYWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func isTTYReader(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
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
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' || r == '/':
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

func ensurePaths(root string) (Paths, error) {
	return ensureInitialized(root)
}

func ensureInitialized(root string) (Paths, error) {
	jotDir := filepath.Join(root, toolDirName)
	topicsDir := filepath.Join(jotDir, topicsDirName)
	statePath := filepath.Join(jotDir, stateFileName)

	if err := os.MkdirAll(topicsDir, 0o755); err != nil {
		return Paths{}, err
	}

	if _, err := os.Stat(statePath); errors.Is(err, os.ErrNotExist) {
		initial := State{Version: 1}
		if err := saveState(statePath, initial); err != nil {
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(path, []byte(""), 0o644)
	} else {
		return err
	}
}

func numberLines(lines []string, color bool) []string {
	width := 2
	if len(lines) > 99 {
		width = len(strconv.Itoa(len(lines)))
	}

	result := make([]string, 0, len(lines))
	for i, line := range lines {
		prefix := fmt.Sprintf("%*d │", width, i+1)
		if color {
			prefix = colorStart + prefix + colorEnd
		}
		result = append(result, prefix+" "+line)
	}
	return result
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

package tools

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	filepath "path/filepath"
	"runtime"
	"strings"

	"github.com/elsejj/gpt/internal/utils"
)

type toolActionType int

const (
	actionOutput toolActionType = iota
	actionCopy
	actionSave
	actionExecute
)

var actionConfirmMessages = map[toolActionType]string{
	actionCopy:    "copy result to clipboard?, [y/N]: ",
	actionSave:    "save result to %q?, [y/N]: ",
	actionExecute: "execute %q?, [y/N]: ",
}

// DoAction executes the configured action for the tool.
// It supports printing to stdout, copying to clipboard, saving to disk and executing commands.
// Placeholders in the action string (e.g. ${name}) are replaced by the provided params and result content.
// When confirmed is false, the user will be asked to confirm before performing non-output actions.
func (tool *Tool) DoAction(content []byte, params map[string]string, confirmed bool) error {
	if tool == nil {
		return fmt.Errorf("tool is nil")
	}

	action := strings.TrimSpace(tool.Action)
	action = utils.ExpandVariables(action, params)
	if action == "" {
		action = "output"
	}

	actionType, target := classifyAction(action)
	if actionType != actionOutput && !confirmed {
		ok, err := confirmAction(actionType, target, content)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("action %q cancelled by user", action)
		}
	}

	switch actionType {
	case actionOutput:
		_, err := os.Stdout.Write(content)
		return err
	case actionCopy:
		return copyToClipboard(content)
	case actionSave:
		return saveToFile(target, content)
	case actionExecute:
		return executeCommand(target, content)
	default:
		return fmt.Errorf("unsupported action %q", action)
	}
}

func classifyAction(action string) (toolActionType, string) {
	clean := strings.TrimSpace(action)
	switch strings.ToLower(clean) {
	case "", "output":
		return actionOutput, ""
	case "copy":
		return actionCopy, ""
	case "execute":
	case "exec":
	case "run":
		return actionExecute, ""
	}

	if clean != "" && !strings.ContainsRune(clean, '\n') &&
		!strings.ContainsAny(clean, " \t") && filepath.Ext(clean) != "" {
		return actionSave, clean
	}
	return actionExecute, clean
}

func confirmAction(actionType toolActionType, action string, content []byte) (bool, error) {
	message, ok := actionConfirmMessages[actionType]
	if !ok {
		return false, fmt.Errorf("no confirm message for action type %d", actionType)
	}
	if actionType == actionExecute && strings.TrimSpace(action) == "" {
		action = string(content)
	}
	fmt.Fprintf(os.Stderr, message, action)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer = strings.TrimSpace(answer)
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes"), nil
}

func copyToClipboard(content []byte) error {
	type candidate struct {
		cmd  string
		args []string
	}

	var attempts []candidate
	switch runtime.GOOS {
	case "darwin":
		attempts = append(attempts, candidate{cmd: "pbcopy"})
	case "windows":
		attempts = append(attempts, candidate{cmd: "clip.exe"})
		attempts = append(attempts, candidate{cmd: "powershell", args: []string{"-NoLogo", "-NoProfile", "-Command", "Set-Clipboard -Value ([Console]::In.ReadToEnd())"}})
	default:
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			attempts = append(attempts, candidate{cmd: "wl-copy"})
		}
		attempts = append(attempts,
			candidate{cmd: "xclip", args: []string{"-selection", "clipboard"}},
			candidate{cmd: "xsel", args: []string{"--clipboard", "--input"}},
		)
	}

	var errs []string
	for _, attempt := range attempts {
		if attempt.cmd == "" || !commandExists(attempt.cmd) {
			continue
		}
		if err := runCommandWithInput(attempt.cmd, attempt.args, content, false); err == nil {
			slog.Info("copied result to clipboard", "command", attempt.cmd)
			return nil
		} else {
			errs = append(errs, fmt.Sprintf("%s: %v", attempt.cmd, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("copy to clipboard failed: %s", strings.Join(errs, "; "))
	}
	return fmt.Errorf("no clipboard utility available")
}

func saveToFile(path string, content []byte) error {
	resolved, err := expandPath(path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(resolved)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(resolved, content, 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", resolved, err)
	}
	slog.Info("saved result to file", "path", resolved)
	return nil
}

func expandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if len(path) == 1 {
			return home, nil
		}
		if path[1] == '/' || path[1] == '\\' {
			return filepath.Join(home, path[2:]), nil
		}
	}
	return filepath.Clean(path), nil
}

func executeCommand(action string, content []byte) error {
	var cmd *exec.Cmd
	if action == "" {
		cmd = buildCommand(string(content))
	} else {
		cmd = buildCommand(action)
		cmd.Stdin = bytes.NewReader(content)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println(cmd.String())
	return cmd.Run()
}

func buildCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("powershell", "-NoLogo", "-NoProfile", "-Command", command)
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return exec.Command(shell, "-c", command)
}

func runCommandWithInput(cmd string, args []string, input []byte, inheritOutput bool) error {
	command := exec.Command(cmd, args...)
	command.Stdin = bytes.NewReader(input)
	if inheritOutput {
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
	}
	return command.Run()
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

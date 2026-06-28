package system

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func CodexPath() (string, error) {
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("codex.cmd"); err == nil {
			return p, nil
		}
	}
	return exec.LookPath("codex")
}

func CodexVersion() string {
	path, err := CodexPath()
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

func CodexConfigPath() string {
	if v := strings.TrimSpace(os.Getenv("CODEX_HOME")); v != "" {
		return filepath.Join(v, "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex", "config.toml")
}

func CodexConfiguredModel() (string, error) {
	path := CodexConfigPath()
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	model, err := rootTOMLString(b, "model")
	if err != nil {
		return "", err
	}
	if !ConcreteCodexModel(model) {
		return "", nil
	}
	return model, nil
}

func ConcreteCodexModel(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "", "default", "codex-default", "local-codex-config", "codex-local-config", "unknown", "unknown-codex-model":
		return false
	default:
		return true
	}
}

func rootTOMLString(b []byte, want string) (string, error) {
	section := ""
	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := stripTOMLComment(strings.TrimSpace(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}
		if section != "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != want {
			continue
		}
		s, err := strconv.Unquote(strings.TrimSpace(value))
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(s), nil
	}
	return "", scanner.Err()
}

func stripTOMLComment(line string) string {
	inString := false
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inString {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if r == '#' && !inString {
			return strings.TrimSpace(line[:i])
		}
	}
	return line
}

func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

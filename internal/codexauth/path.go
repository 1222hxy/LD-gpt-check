package codexauth

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type AuthPathCandidate struct {
	Path   string
	Source string
}

func ResolveAuthPath(explicit string) string {
	if explicit = strings.TrimSpace(explicit); explicit != "" {
		return explicit
	}
	if home := strings.TrimSpace(os.Getenv("CODEX_HOME")); home != "" {
		return filepath.Join(home, "auth.json")
	}
	candidates := CandidateAuthPaths("")
	for _, c := range candidates {
		if fileExists(c.Path) {
			return c.Path
		}
	}
	if len(candidates) > 0 {
		return candidates[0].Path
	}
	return ""
}

func CandidateAuthPaths(explicit string) []AuthPathCandidate {
	if explicit = strings.TrimSpace(explicit); explicit != "" {
		return []AuthPathCandidate{{Path: explicit, Source: "explicit"}}
	}
	var out []AuthPathCandidate
	if home := strings.TrimSpace(os.Getenv("CODEX_HOME")); home != "" {
		out = append(out, AuthPathCandidate{Path: filepath.Join(home, "auth.json"), Source: "CODEX_HOME"})
	}
	userHome, err := os.UserHomeDir()
	if err == nil && userHome != "" {
		out = append(out, AuthPathCandidate{Path: filepath.Join(userHome, ".codex", "auth.json"), Source: "Codex CLI"})
		out = append(out, appAuthPathCandidates(userHome)...)
	}
	out = append(out, envAppDataCandidates()...)
	return dedupeAuthPathCandidates(out)
}

func appAuthPathCandidates(home string) []AuthPathCandidate {
	switch runtime.GOOS {
	case "windows":
		return []AuthPathCandidate{
			{Path: filepath.Join(home, "AppData", "Roaming", "Codex", "auth.json"), Source: "Codex App"},
			{Path: filepath.Join(home, "AppData", "Roaming", "OpenAI", "Codex", "auth.json"), Source: "Codex App"},
			{Path: filepath.Join(home, "AppData", "Roaming", "com.openai.codex", "auth.json"), Source: "Codex App"},
			{Path: filepath.Join(home, "AppData", "Local", "Codex", "auth.json"), Source: "Codex App"},
			{Path: filepath.Join(home, "AppData", "Local", "OpenAI", "Codex", "auth.json"), Source: "Codex App"},
			{Path: filepath.Join(home, "AppData", "Local", "com.openai.codex", "auth.json"), Source: "Codex App"},
		}
	case "darwin":
		return []AuthPathCandidate{
			{Path: filepath.Join(home, "Library", "Application Support", "Codex", "auth.json"), Source: "Codex App"},
			{Path: filepath.Join(home, "Library", "Application Support", "OpenAI", "Codex", "auth.json"), Source: "Codex App"},
			{Path: filepath.Join(home, "Library", "Application Support", "com.openai.codex", "auth.json"), Source: "Codex App"},
		}
	default:
		return []AuthPathCandidate{
			{Path: filepath.Join(home, ".config", "Codex", "auth.json"), Source: "Codex App"},
			{Path: filepath.Join(home, ".config", "OpenAI", "Codex", "auth.json"), Source: "Codex App"},
			{Path: filepath.Join(home, ".config", "openai-codex", "auth.json"), Source: "Codex App"},
			{Path: filepath.Join(home, ".config", "com.openai.codex", "auth.json"), Source: "Codex App"},
		}
	}
}

func envAppDataCandidates() []AuthPathCandidate {
	var out []AuthPathCandidate
	for _, env := range []string{"APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME"} {
		base := strings.TrimSpace(os.Getenv(env))
		if base == "" {
			continue
		}
		out = append(out,
			AuthPathCandidate{Path: filepath.Join(base, "Codex", "auth.json"), Source: env},
			AuthPathCandidate{Path: filepath.Join(base, "OpenAI", "Codex", "auth.json"), Source: env},
			AuthPathCandidate{Path: filepath.Join(base, "com.openai.codex", "auth.json"), Source: env},
		)
	}
	return out
}

func dedupeAuthPathCandidates(in []AuthPathCandidate) []AuthPathCandidate {
	seen := make(map[string]bool, len(in))
	out := make([]AuthPathCandidate, 0, len(in))
	for _, c := range in {
		c.Path = strings.TrimSpace(c.Path)
		if c.Path == "" {
			continue
		}
		key := filepath.Clean(c.Path)
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}
	return out
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

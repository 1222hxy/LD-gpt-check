package codexauth

import (
	"encoding/json"
	"os"
)

func LoadRaw(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func Load(path string) (*CodexAuth, error) {
	path = ResolveAuthPath(path)
	raw, err := LoadRaw(path)
	if err != nil {
		return nil, err
	}
	return Parse(raw, path), nil
}

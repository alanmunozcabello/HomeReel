package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	MediaDir string `json:"media_dir"`
}

func Load(path string) (Settings, error) {
	var settings Settings
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	return settings, json.Unmarshal(data, &settings)
}

func Save(path string, settings Settings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

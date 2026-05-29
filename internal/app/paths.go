package app

import "path/filepath"

func ComponentsPath(configDir string) string {
	return filepath.Join(configDir, "raven", "components.json")
}

func EventsPath(configDir string) string {
	return filepath.Join(configDir, "raven", "events.json")
}

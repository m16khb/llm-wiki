package vault

import (
	"fmt"
	"os"
	"strings"
)

const EnvVar = "LLM_WIKI_VAULT"

func Resolve(path string) (string, error) {
	return ResolveWithDefault(path, "")
}

func ResolveWithDefault(path string, defaultPath string) (string, error) {
	if strings.TrimSpace(path) != "" {
		return path, nil
	}
	if strings.TrimSpace(defaultPath) != "" {
		return defaultPath, nil
	}
	configured := strings.TrimSpace(os.Getenv(EnvVar))
	if configured != "" {
		return configured, nil
	}
	return "", fmt.Errorf("path is required unless %s is set", EnvVar)
}

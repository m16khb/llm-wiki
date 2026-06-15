package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/m16khb/llm-wiki/internal/okf"
)

type Event struct {
	Event   string `json:"event"`
	Host    string `json:"host"`
	Message string `json:"message,omitempty"`
}

type AppendResult struct {
	OK     bool   `json:"ok"`
	Path   string `json:"path"`
	Event  string `json:"event"`
	Host   string `json:"host"`
	Logged bool   `json:"logged"`
}

var secretRE = regexp.MustCompile(`(?i)(token|api[_-]?key|secret|password)=([^\\s"']+)`)

func AppendEvent(root string, event Event) (AppendResult, error) {
	dir, err := okf.SafeWritePath(root, ".llm-wiki")
	if err != nil {
		return AppendResult{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return AppendResult{}, err
	}
	path := filepath.Join(dir, "hooks.jsonl")
	lock := flock.New(path + ".lock")
	if err := lock.Lock(); err != nil {
		return AppendResult{}, err
	}
	defer lock.Unlock()
	record := map[string]any{
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"event":   event.Event,
		"host":    event.Host,
		"message": capString(redact(event.Message), 1800),
	}
	b, err := json.Marshal(record)
	if err != nil {
		return AppendResult{}, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return AppendResult{}, err
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		return AppendResult{}, err
	}
	return AppendResult{OK: true, Path: path, Event: event.Event, Host: event.Host, Logged: true}, nil
}

func OutputForHost(host, event, decision string) map[string]any {
	switch decision {
	case "deny":
		if host == "codex" {
			return map[string]any{"decision": "block", "reason": "llm-wiki hook denied the action"}
		}
		return map[string]any{"hookSpecificOutput": map[string]any{
			"hookEventName":            event,
			"permissionDecision":       "deny",
			"permissionDecisionReason": "llm-wiki hook denied the action",
		}}
	case "ask":
		return map[string]any{"hookSpecificOutput": map[string]any{
			"hookEventName":            event,
			"permissionDecision":       "ask",
			"permissionDecisionReason": "llm-wiki hook requests confirmation",
		}}
	case "block":
		if event == "Stop" {
			return map[string]any{"continue": true, "decision": "block", "reason": "llm-wiki hook requested continuation"}
		}
	}
	return map[string]any{}
}

func redact(value string) string {
	return secretRE.ReplaceAllString(value, "$1=[REDACTED]")
}

func capString(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "...[truncated]"
}

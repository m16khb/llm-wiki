package logstore

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/m16khb/llm-wiki/internal/okf"
)

type AppendResult struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path"`
	Bytes   int    `json:"bytes"`
	Op      string `json:"op"`
	Message string `json:"message"`
}

func Append(root, op, message string) (AppendResult, error) {
	path, err := okf.SafeWritePath(root, "log.md")
	if err != nil {
		return AppendResult{}, err
	}
	lock := flock.New(path + ".lock")
	if err := lock.Lock(); err != nil {
		return AppendResult{}, err
	}
	defer lock.Unlock()
	entry := fmt.Sprintf("- op: %s\n  at: %s\n  message: %s\n", sanitizeLine(op), time.Now().UTC().Format(time.RFC3339), sanitizeLine(message))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return AppendResult{}, err
	}
	defer f.Close()
	n, err := f.WriteString(entry)
	if err != nil {
		return AppendResult{}, err
	}
	return AppendResult{OK: true, Path: path, Bytes: n, Op: op, Message: message}, nil
}

func sanitizeLine(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.TrimSpace(value)
}

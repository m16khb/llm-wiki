package daemon

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	message          = "daemon runtime is not implemented; use llm-wiki mcp for direct stdio MCP"
	warning          = "daemon runtime is not implemented"
	socketName       = "daemon.sock"
	pidName          = "daemon.pid"
	lockName         = "daemon.lock"
	stateDirEnv      = "LLM_WIKI_STATE_DIR"
	xdgStateHomeEnv  = "XDG_STATE_HOME"
	defaultStatePath = ".local/state/llm-wiki"
)

var ErrUnsupported = errors.New("daemon runtime is not implemented")

type Paths struct {
	StateDir   string
	SocketPath string
	PIDPath    string
	LockPath   string
}

type Result struct {
	OK          bool     `json:"ok"`
	Action      string   `json:"action"`
	Implemented bool     `json:"implemented"`
	Running     bool     `json:"running"`
	StateDir    string   `json:"state_dir"`
	SocketPath  string   `json:"socket_path"`
	PIDPath     string   `json:"pid_path"`
	LockPath    string   `json:"lock_path"`
	Message     string   `json:"message"`
	Warnings    []string `json:"warnings"`
}

func ResolvePaths() (Paths, error) {
	stateDir := os.Getenv(stateDirEnv)
	if stateDir == "" {
		if xdg := os.Getenv(xdgStateHomeEnv); xdg != "" {
			stateDir = filepath.Join(xdg, "llm-wiki")
		}
	}
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, err
		}
		stateDir = filepath.Join(home, defaultStatePath)
	}

	stateDir, err := filepath.Abs(stateDir)
	if err != nil {
		return Paths{}, err
	}
	return Paths{
		StateDir:   stateDir,
		SocketPath: filepath.Join(stateDir, socketName),
		PIDPath:    filepath.Join(stateDir, pidName),
		LockPath:   filepath.Join(stateDir, lockName),
	}, nil
}

func Status() (Result, error) {
	return result("status", true, nil)
}

func Doctor() (Result, error) {
	return result("doctor", true, []string{warning})
}

func Start() (Result, error) {
	r, err := result("start", false, []string{warning})
	if err != nil {
		return r, err
	}
	return r, ErrUnsupported
}

func Stop() (Result, error) {
	r, err := result("stop", false, []string{warning})
	if err != nil {
		return r, err
	}
	return r, ErrUnsupported
}

func result(action string, ok bool, warnings []string) (Result, error) {
	paths, err := ResolvePaths()
	if err != nil {
		return Result{}, err
	}
	if warnings == nil {
		warnings = []string{}
	}
	return Result{
		OK:          ok,
		Action:      action,
		Implemented: false,
		Running:     false,
		StateDir:    paths.StateDir,
		SocketPath:  paths.SocketPath,
		PIDPath:     paths.PIDPath,
		LockPath:    paths.LockPath,
		Message:     message,
		Warnings:    warnings,
	}, nil
}

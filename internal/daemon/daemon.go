package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gofrs/flock"
	wikimcp "github.com/m16khb/llm-wiki/internal/mcp"
)

const (
	runningMessage   = "daemon is running"
	stoppedMessage   = "daemon is stopped"
	socketName       = "daemon.sock"
	pidName          = "daemon.pid"
	lockName         = "daemon.lock"
	logName          = "daemon.log"
	stateDirEnv      = "LLM_WIKI_STATE_DIR"
	xdgStateHomeEnv  = "XDG_STATE_HOME"
	defaultStatePath = ".local/state/llm-wiki"
	readyTimeout     = 15 * time.Second
	maxConnections   = 64
	staleStopTimeout = time.Second
)

var (
	currentExecutable     = os.Executable
	findSiblingDaemonPIDs = defaultFindSiblingDaemonPIDs
	daemonStateDirForPID  = defaultDaemonStateDirForPID
	processAlivePID       = processAlive
	signalPID             = signalProcess
	killPID               = killProcess
)

type Paths struct {
	StateDir   string
	SocketPath string
	PIDPath    string
	LockPath   string
	LogPath    string
}

type Result struct {
	OK          bool     `json:"ok"`
	Action      string   `json:"action"`
	Implemented bool     `json:"implemented"`
	Running     bool     `json:"running"`
	PID         int      `json:"pid,omitempty"`
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
		LogPath:    filepath.Join(stateDir, logName),
	}, nil
}

func Status() (Result, error) {
	return statusResult("status")
}

func Doctor() (Result, error) {
	result, err := statusResult("doctor")
	if err != nil {
		return result, err
	}
	if !result.Running {
		result.Warnings = append(result.Warnings, "daemon is not running; llm-wiki mcp will auto-start it")
	}
	return result, nil
}

func Start() (Result, error) {
	return EnsureRunning()
}

func EnsureRunning() (Result, error) {
	if result, err := statusResult("start"); err == nil && result.Running {
		result.Warnings = append(result.Warnings, cleanupSiblingDaemons(result.StateDir, result.PID)...)
		return result, nil
	}
	paths, err := ResolvePaths()
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(paths.StateDir, 0o700); err != nil {
		return Result{}, err
	}
	lock := flock.New(paths.LockPath)
	ctx, cancel := context.WithTimeout(context.Background(), readyTimeout)
	defer cancel()
	locked, err := lock.TryLockContext(ctx, 50*time.Millisecond)
	if err != nil {
		return Result{}, err
	}
	if !locked {
		return waitForRunning("start", paths, readyTimeout)
	}
	defer func() {
		_ = lock.Unlock()
		_ = os.Remove(paths.LockPath)
	}()
	if result, err := statusResult("start"); err == nil && result.Running {
		result.Warnings = append(result.Warnings, cleanupSiblingDaemons(result.StateDir, result.PID)...)
		return result, nil
	}
	_ = os.Remove(paths.SocketPath)
	exe, err := currentExecutable()
	if err != nil {
		return Result{}, err
	}
	logFile, err := os.OpenFile(paths.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Result{}, err
	}
	defer logFile.Close()
	cmd := daemonServerCommand(exe, paths, logFile)
	if err := cmd.Start(); err != nil {
		return Result{OK: false, Action: "start", Implemented: true, StateDir: paths.StateDir, SocketPath: paths.SocketPath, PIDPath: paths.PIDPath, LockPath: paths.LockPath, Message: err.Error()}, err
	}
	_ = cmd.Process.Release()
	result, err := waitForRunning("start", paths, readyTimeout)
	if err != nil {
		return result, err
	}
	result.Warnings = append(result.Warnings, cleanupSiblingDaemons(result.StateDir, result.PID)...)
	return result, nil
}

func daemonServerCommand(exe string, paths Paths, logFile *os.File) *exec.Cmd {
	cmd := exec.Command(exe, "daemon", "--internal")
	cmd.Env = append(os.Environ(), stateDirEnv+"="+paths.StateDir)
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd
}

func Stop() (Result, error) {
	result, err := statusResult("stop")
	if err != nil {
		return result, err
	}
	if !result.Running {
		result.Warnings = append(result.Warnings, cleanupSiblingDaemons(result.StateDir, 0)...)
		_ = os.Remove(result.SocketPath)
		_ = os.Remove(result.PIDPath)
		_ = os.Remove(result.LockPath)
		result.OK = true
		result.PID = 0
		result.Message = stoppedMessage
		return result, nil
	}
	proc, err := os.FindProcess(result.PID)
	if err != nil {
		return result, err
	}
	_ = proc.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(readyTimeout)
	for time.Now().Before(deadline) {
		current, err := statusResult("stop")
		if err != nil {
			return current, err
		}
		if !current.Running {
			current.Warnings = append(current.Warnings, cleanupSiblingDaemons(current.StateDir, 0)...)
			current.OK = true
			current.Message = stoppedMessage
			return current, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = proc.Kill()
	result.Warnings = append(result.Warnings, cleanupSiblingDaemons(result.StateDir, 0)...)
	_ = os.Remove(result.SocketPath)
	_ = os.Remove(result.PIDPath)
	result.Running = false
	result.OK = true
	result.Message = stoppedMessage
	return result, nil
}

func cleanupSiblingDaemons(stateDir string, keepPID int) []string {
	exe, err := currentExecutable()
	if err != nil {
		return []string{fmt.Sprintf("could not inspect daemon siblings: %v", err)}
	}
	pids, err := findSiblingDaemonPIDs(exe)
	if err != nil {
		return []string{fmt.Sprintf("could not inspect daemon siblings: %v", err)}
	}
	warnings := []string{}
	for _, pid := range pids {
		if pid <= 0 || pid == os.Getpid() || pid == keepPID {
			continue
		}
		pidStateDir, ok := daemonStateDirForPID(pid)
		if !ok || pidStateDir != stateDir {
			continue
		}
		if err := stopProcess(pid, staleStopTimeout); err != nil {
			warnings = append(warnings, fmt.Sprintf("could not stop stale daemon pid %d: %v", pid, err))
			continue
		}
		warnings = append(warnings, fmt.Sprintf("stopped stale daemon pid %d", pid))
	}
	return warnings
}

func stopProcess(pid int, timeout time.Duration) error {
	if err := signalPID(pid, syscall.SIGTERM); err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processAlivePID(pid) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := killPID(pid); err != nil {
		return err
	}
	return nil
}

func defaultFindSiblingDaemonPIDs(exe string) ([]int, error) {
	out, err := exec.Command("pgrep", "-f", regexp.QuoteMeta(exe)+" daemon --internal").Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
			return nil, nil
		}
		return nil, err
	}
	return parsePIDs(string(out)), nil
}

func defaultDaemonStateDirForPID(pid int) (string, bool) {
	out, err := exec.Command("ps", "eww", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		return "", false
	}
	for _, field := range strings.Fields(string(out)) {
		if strings.HasPrefix(field, stateDirEnv+"=") {
			return strings.TrimPrefix(field, stateDirEnv+"="), true
		}
	}
	return "", false
}

func parsePIDs(output string) []int {
	lines := strings.Fields(output)
	pids := make([]int, 0, len(lines))
	for _, line := range lines {
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids
}

func signalProcess(pid int, signal syscall.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(signal)
}

func killProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}

func RunServer(ctx context.Context) error {
	paths, err := ResolvePaths()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(paths.StateDir, 0o700); err != nil {
		return err
	}
	logFile, err := os.OpenFile(paths.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer logFile.Close()
	_ = os.Remove(paths.SocketPath)
	listener, err := net.Listen("unix", paths.SocketPath)
	if err != nil {
		return err
	}
	defer listener.Close()
	_ = os.Chmod(paths.SocketPath, 0o600)
	if err := os.WriteFile(paths.PIDPath, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		return err
	}
	_ = os.Remove(paths.LockPath)
	defer func() {
		_ = os.Remove(paths.SocketPath)
		_ = os.Remove(paths.PIDPath)
	}()
	fmt.Fprintf(logFile, "%s daemon started pid=%d socket=%s\n", time.Now().UTC().Format(time.RFC3339), os.Getpid(), paths.SocketPath)
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		_ = listener.Close()
		close(done)
	}()
	err = acceptLoop(listener, logFile)
	select {
	case <-done:
	default:
	}
	fmt.Fprintf(logFile, "%s daemon stopped\n", time.Now().UTC().Format(time.RFC3339))
	if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		return err
	}
	return nil
}

func RunMCPProxy() error {
	status, err := EnsureRunning()
	if err != nil {
		return err
	}
	conn, err := net.Dial("unix", status.SocketPath)
	if err != nil {
		return fmt.Errorf("connect daemon: %w", err)
	}
	defer conn.Close()
	stdoutDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(os.Stdout, conn)
		stdoutDone <- err
	}()
	stdinDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(conn, os.Stdin)
		if unixConn, ok := any(conn).(interface{ CloseWrite() error }); ok {
			_ = unixConn.CloseWrite()
		}
		stdinDone <- err
	}()
	select {
	case stdinErr := <-stdinDone:
		stdoutErr := <-stdoutDone
		if stdinErr != nil && !errors.Is(stdinErr, net.ErrClosed) {
			return stdinErr
		}
		return proxyOutputError(stdoutErr)
	case stdoutErr := <-stdoutDone:
		_ = conn.Close()
		return proxyOutputError(stdoutErr)
	}
}

func acceptLoop(listener net.Listener, logFile io.Writer) error {
	connSlots := make(chan struct{}, maxConnections)
	var active sync.WaitGroup
	defer active.Wait()
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		select {
		case connSlots <- struct{}{}:
		default:
			_, _ = conn.Write([]byte("daemon connection limit reached\n"))
			_ = conn.Close()
			continue
		}
		active.Add(1)
		go func(conn net.Conn) {
			defer func() {
				<-connSlots
				active.Done()
			}()
			defer conn.Close()
			if err := wikimcp.RunStream(context.Background(), conn); err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				fmt.Fprintf(logFile, "%s mcp stream error: %v\n", time.Now().UTC().Format(time.RFC3339), err)
			}
		}(conn)
	}
}

func statusResult(action string) (Result, error) {
	paths, err := ResolvePaths()
	if err != nil {
		return Result{}, err
	}
	pid := readPID(paths.PIDPath)
	running := pid > 0 && processAlive(pid) && socketReachable(paths.SocketPath)
	message := stoppedMessage
	if running {
		message = runningMessage
	} else {
		pid = 0
	}
	return Result{
		OK:          true,
		Action:      action,
		Implemented: true,
		Running:     running,
		PID:         pid,
		StateDir:    paths.StateDir,
		SocketPath:  paths.SocketPath,
		PIDPath:     paths.PIDPath,
		LockPath:    paths.LockPath,
		Message:     message,
		Warnings:    []string{},
	}, nil
}

func waitForRunning(action string, paths Paths, timeout time.Duration) (Result, error) {
	deadline := time.Now().Add(timeout)
	var last Result
	for time.Now().Before(deadline) {
		result, err := statusResult(action)
		if err != nil {
			return result, err
		}
		last = result
		if result.Running {
			return result, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	if last.StateDir == "" {
		last = Result{OK: false, Action: action, Implemented: true, StateDir: paths.StateDir, SocketPath: paths.SocketPath, PIDPath: paths.PIDPath, LockPath: paths.LockPath}
	}
	last.OK = false
	last.Message = "daemon did not become ready before timeout"
	return last, errors.New(last.Message)
}

func readPID(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func socketReachable(path string) bool {
	conn, err := net.DialTimeout("unix", path, 100*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func proxyOutputError(err error) error {
	if err != nil && !errors.Is(err, net.ErrClosed) && !strings.Contains(err.Error(), "use of closed network connection") {
		return err
	}
	return nil
}

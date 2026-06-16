package daemon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/m16khb/llm-wiki/internal/vault"
)

const (
	daemonProtocol   = "llm-wiki-daemon/1"
	runningMessage   = "daemon is running"
	stoppedMessage   = "daemon is stopped"
	socketName       = "daemon.sock"
	pidName          = "daemon.pid"
	lockName         = "daemon.lock"
	logName          = "daemon.log"
	metaName         = "daemon.meta.json"
	stateDirEnv      = "LLM_WIKI_STATE_DIR"
	xdgStateHomeEnv  = "XDG_STATE_HOME"
	defaultStatePath = ".local/state/llm-wiki"
	readyTimeout     = 15 * time.Second
	maxConnections   = 64
	staleStopTimeout = time.Second
	metaVersion      = 1
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
	MetaPath   string
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

type daemonFrame struct {
	Protocol  string `json:"protocol"`
	Kind      string `json:"kind"`
	VaultPath string `json:"vault_path,omitempty"`
}

type daemonMeta struct {
	ProtocolVersion int    `json:"protocol_version"`
	PID             int    `json:"pid"`
	ExecutablePath  string `json:"executable_path"`
	ExecutableSize  int64  `json:"executable_size"`
	ExecutableMTime int64  `json:"executable_mtime"`
}

type daemonCompatibility int

const (
	daemonCompatible daemonCompatibility = iota
	daemonNeedsGracefulReplace
	daemonNeedsHardRestart
)

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
		MetaPath:   filepath.Join(stateDir, metaName),
	}, nil
}

func readDaemonFrame(reader *bufio.Reader) (daemonFrame, []byte, error) {
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return daemonFrame{}, nil, err
	}
	var frame daemonFrame
	if err := json.Unmarshal(bytes.TrimSpace(line), &frame); err != nil || frame.Protocol != daemonProtocol || frame.Kind == "" {
		return daemonFrame{}, line, nil
	}
	return frame, nil, nil
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
	paths, err := ResolvePaths()
	if err != nil {
		return Result{}, err
	}
	if result, done, err := useRunningDaemonIfCompatible("start", paths); err != nil || done {
		return result, err
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
		result, err := waitForRunning("start", paths, readyTimeout)
		if err != nil || !result.Running {
			return result, err
		}
		result, _, err = useRunningDaemonIfCompatible("start", paths)
		return result, err
	}
	defer func() {
		_ = lock.Unlock()
		_ = os.Remove(paths.LockPath)
	}()
	if result, done, err := useRunningDaemonIfCompatibleLocked("start", paths); err != nil || done {
		return result, err
	}
	return startDaemonLocked("start", paths, nil)
}

func Replace() (Result, error) {
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
		result, err := waitForRunning("replace", paths, readyTimeout)
		if err != nil {
			return result, err
		}
		return result, nil
	}
	defer func() {
		_ = lock.Unlock()
		_ = os.Remove(paths.LockPath)
	}()
	result, err := statusResult("replace")
	if err != nil {
		return result, err
	}
	warnings := []string{}
	if result.Running {
		switch compatibility, reason := daemonRuntimeCompatibility(paths, result.PID); compatibility {
		case daemonCompatible, daemonNeedsGracefulReplace:
			if err := drainDaemon(paths.SocketPath); err != nil {
				result.OK = false
				result.Message = err.Error()
				return result, err
			}
			if err := waitForSocketReleased(paths.SocketPath, readyTimeout); err != nil {
				result.OK = false
				result.Message = err.Error()
				return result, err
			}
			warnings = append(warnings, fmt.Sprintf("drained daemon pid %d for replacement", result.PID))
		case daemonNeedsHardRestart:
			warning, err := hardRestartDaemon(result, reason)
			warnings = append(warnings, warning)
			if err != nil {
				result.OK = false
				result.Message = err.Error()
				result.Warnings = append(result.Warnings, warnings...)
				return result, err
			}
		}
	}
	return startDaemonLocked("replace", paths, warnings)
}

func startDaemonLocked(action string, paths Paths, warnings []string) (Result, error) {
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
		return Result{OK: false, Action: action, Implemented: true, StateDir: paths.StateDir, SocketPath: paths.SocketPath, PIDPath: paths.PIDPath, LockPath: paths.LockPath, Message: err.Error()}, err
	}
	_ = cmd.Process.Release()
	result, err := waitForRunning(action, paths, readyTimeout)
	if err != nil {
		return result, err
	}
	existingWarnings := result.Warnings
	result.Warnings = append([]string{}, warnings...)
	result.Warnings = append(result.Warnings, existingWarnings...)
	result.Warnings = append(result.Warnings, cleanupSiblingDaemons(result.StateDir, result.PID)...)
	return result, nil
}

func useRunningDaemonIfCompatible(action string, paths Paths) (Result, bool, error) {
	result, err := statusResult(action)
	if err != nil || !result.Running {
		return result, false, err
	}
	switch compatibility, reason := daemonRuntimeCompatibility(paths, result.PID); compatibility {
	case daemonCompatible:
		result.Warnings = append(result.Warnings, cleanupSiblingDaemons(result.StateDir, result.PID)...)
		return result, true, nil
	case daemonNeedsGracefulReplace:
		replaced, err := Replace()
		return replaced, true, err
	case daemonNeedsHardRestart:
		warning, err := hardRestartDaemon(result, reason)
		result.Warnings = append(result.Warnings, warning)
		if err != nil {
			result.OK = false
			result.Message = err.Error()
			return result, true, err
		}
		return result, false, nil
	default:
		return result, false, nil
	}
}

func useRunningDaemonIfCompatibleLocked(action string, paths Paths) (Result, bool, error) {
	result, err := statusResult(action)
	if err != nil || !result.Running {
		return result, false, err
	}
	switch compatibility, reason := daemonRuntimeCompatibility(paths, result.PID); compatibility {
	case daemonCompatible:
		result.Warnings = append(result.Warnings, cleanupSiblingDaemons(result.StateDir, result.PID)...)
		return result, true, nil
	case daemonNeedsGracefulReplace:
		if err := drainDaemon(paths.SocketPath); err != nil {
			result.OK = false
			result.Message = err.Error()
			return result, true, err
		}
		if err := waitForSocketReleased(paths.SocketPath, readyTimeout); err != nil {
			result.OK = false
			result.Message = err.Error()
			return result, true, err
		}
		return result, false, nil
	case daemonNeedsHardRestart:
		warning, err := hardRestartDaemon(result, reason)
		result.Warnings = append(result.Warnings, warning)
		if err != nil {
			result.OK = false
			result.Message = err.Error()
			return result, true, err
		}
		return result, false, nil
	default:
		return result, false, nil
	}
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

func daemonRuntimeCompatibility(paths Paths, pid int) (daemonCompatibility, string) {
	meta, err := readDaemonMeta(paths.MetaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return daemonNeedsHardRestart, "metadata is missing"
		}
		return daemonNeedsHardRestart, fmt.Sprintf("metadata cannot be read: %v", err)
	}
	if meta.ProtocolVersion != metaVersion {
		return daemonNeedsHardRestart, fmt.Sprintf("metadata protocol version is %d", meta.ProtocolVersion)
	}
	if meta.PID != pid {
		return daemonNeedsHardRestart, fmt.Sprintf("metadata pid is %d", meta.PID)
	}
	identity, err := currentExecutableIdentity(pid)
	if err != nil {
		return daemonNeedsHardRestart, fmt.Sprintf("current executable cannot be inspected: %v", err)
	}
	if meta.ExecutablePath != identity.ExecutablePath || meta.ExecutableSize != identity.ExecutableSize || meta.ExecutableMTime != identity.ExecutableMTime {
		return daemonNeedsGracefulReplace, "executable identity changed"
	}
	return daemonCompatible, ""
}

func hardRestartDaemon(result Result, reason string) (string, error) {
	warning := fmt.Sprintf("stopped daemon pid %d because %s", result.PID, reason)
	if err := stopProcess(result.PID, staleStopTimeout); err != nil {
		return warning, fmt.Errorf("restart daemon: %w", err)
	}
	releaseRuntimeFiles(Paths{StateDir: result.StateDir, SocketPath: result.SocketPath, PIDPath: result.PIDPath, MetaPath: filepath.Join(result.StateDir, metaName)}, result.PID)
	return warning, nil
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
		if drainingMarkerExists(stateDir, pid) {
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
	return envVarForPID(pid, stateDirEnv)
}

func envVarForPID(pid int, name string) (string, bool) {
	out, err := exec.Command("ps", "eww", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		return "", false
	}
	for _, field := range strings.Fields(string(out)) {
		if strings.HasPrefix(field, name+"=") {
			return strings.TrimPrefix(field, name+"="), true
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
	if err := writeDaemonMeta(paths); err != nil {
		return err
	}
	_ = os.Remove(paths.LockPath)
	defer func() {
		releaseRuntimeFiles(paths, os.Getpid())
		_ = os.Remove(drainingMarkerPath(paths.StateDir, os.Getpid()))
	}()
	fmt.Fprintf(logFile, "%s daemon started pid=%d socket=%s\n", time.Now().UTC().Format(time.RFC3339), os.Getpid(), paths.SocketPath)
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		_ = listener.Close()
		close(done)
	}()
	err = acceptLoop(listener, logFile, paths)
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
	if err := writeDaemonFrame(conn, daemonFrame{Kind: "mcp", VaultPath: strings.TrimSpace(os.Getenv(vault.EnvVar))}); err != nil {
		return err
	}
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

func acceptLoop(listener net.Listener, logFile io.Writer, paths Paths) error {
	connSlots := make(chan struct{}, maxConnections)
	var active sync.WaitGroup
	var drainOnce sync.Once
	drain := func() {
		drainOnce.Do(func() {
			_ = os.WriteFile(drainingMarkerPath(paths.StateDir, os.Getpid()), []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600)
			releaseRuntimeFiles(paths, os.Getpid())
			_ = listener.Close()
		})
	}
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
			if err := handleDaemonConnection(conn, drain); err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				fmt.Fprintf(logFile, "%s mcp stream error: %v\n", time.Now().UTC().Format(time.RFC3339), err)
			}
		}(conn)
	}
}

func handleDaemonConnection(conn net.Conn, drain func()) error {
	reader := bufio.NewReader(conn)
	frame, replay, err := readDaemonFrame(reader)
	if err != nil {
		return err
	}
	if frame.Kind == "drain" {
		drain()
		return nil
	}
	ctx := context.Background()
	if frame.Kind == "mcp" {
		ctx = wikimcp.WithDefaultVault(ctx, frame.VaultPath)
		return wikimcp.RunStream(ctx, &readerConn{Conn: conn, reader: reader})
	}
	return wikimcp.RunStream(ctx, &readerConn{Conn: conn, reader: io.MultiReader(bytes.NewReader(replay), reader)})
}

type readerConn struct {
	net.Conn
	reader io.Reader
}

func (c *readerConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
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

func writeDaemonFrame(w io.Writer, frame daemonFrame) error {
	frame.Protocol = daemonProtocol
	data, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	_, err = w.Write(append(data, '\n'))
	return err
}

func drainDaemon(socketPath string) error {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("connect daemon for drain: %w", err)
	}
	defer conn.Close()
	if err := writeDaemonFrame(conn, daemonFrame{Kind: "drain"}); err != nil {
		return fmt.Errorf("send drain frame: %w", err)
	}
	return nil
}

func waitForSocketReleased(socketPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); os.IsNotExist(err) {
			return nil
		}
		if !socketReachable(socketPath) {
			_ = os.Remove(socketPath)
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("daemon socket was not released before timeout")
}

func writeDaemonMeta(paths Paths) error {
	identity, err := currentExecutableIdentity(os.Getpid())
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(paths.MetaPath, append(data, '\n'), 0o600)
}

func readDaemonMeta(path string) (daemonMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return daemonMeta{}, err
	}
	var meta daemonMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return daemonMeta{}, err
	}
	return meta, nil
}

func currentExecutableIdentity(pid int) (daemonMeta, error) {
	exe, err := currentExecutable()
	if err != nil {
		return daemonMeta{}, err
	}
	info, err := os.Stat(exe)
	if err != nil {
		return daemonMeta{}, err
	}
	return daemonMeta{
		ProtocolVersion: metaVersion,
		PID:             pid,
		ExecutablePath:  exe,
		ExecutableSize:  info.Size(),
		ExecutableMTime: info.ModTime().UnixNano(),
	}, nil
}

func releaseRuntimeFiles(paths Paths, pid int) {
	if readPID(paths.PIDPath) != pid {
		return
	}
	_ = os.Remove(paths.SocketPath)
	_ = os.Remove(paths.PIDPath)
	_ = os.Remove(paths.MetaPath)
}

func drainingMarkerPath(stateDir string, pid int) string {
	return filepath.Join(stateDir, fmt.Sprintf("daemon.%d.draining", pid))
}

func drainingMarkerExists(stateDir string, pid int) bool {
	_, err := os.Stat(drainingMarkerPath(stateDir, pid))
	return err == nil
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

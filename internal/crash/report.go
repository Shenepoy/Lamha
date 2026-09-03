// Package crash records local diagnostics when Lamha does not exit normally.
//
// Reports stay on the user's machine. The UI can turn a report into a
// pre-filled GitHub issue, but submitting it always remains an explicit user
// action.
package crash

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	logFileName     = "lamha.log"
	lockFileName    = "crash.lock"
	markerFileName  = "running.json"
	reportFileName  = "crash-report.txt"
	pendingFileName = "crash-pending"

	maxLogBytes         = 512 * 1024
	maxReportBytes      = 32 * 1024
	maxIssueReportBytes = 12 * 1024
)

// ErrAlreadyRunning means another Lamha process owns the crash-tracking lock.
// This is expected for a forwarded GApplication command-line invocation.
var ErrAlreadyRunning = errors.New("another Lamha process owns crash tracking")

// AppInfo identifies the running Lamha process in a report.
type AppInfo struct {
	Version   string
	PID       int
	StartedAt time.Time
}

// Report is a locally saved crash report waiting for the user's review.
type Report struct {
	Text string
	Path string
}

type sessionMarker struct {
	Version     string      `json:"version"`
	PID         int         `json:"pid"`
	StartedAt   time.Time   `json:"started_at"`
	Environment environment `json:"environment"`
}

// Session is the lifecycle handle for crash tracking.
type Session struct {
	mu sync.Mutex

	lockFile    *os.File
	markerPath  string
	logPath     string
	reportPath  string
	pendingPath string
	info        AppInfo
	env         environment
	logFile     *os.File
	closed      bool
}

// Start initializes crash tracking in the user's state directory. It returns
// a pending report from an earlier unexpected exit, if one exists.
//
// The caller should set the standard logger to LogFile and call Close when the
// application exits normally. Tracking is best-effort; an unavailable state
// directory is returned as an error so the caller can continue without it.
// ErrAlreadyRunning is returned for a secondary process forwarded to the
// primary GApplication instance.
func Start(info AppInfo) (*Session, *Report, error) {
	dir, err := stateDir()
	if err != nil {
		return nil, nil, err
	}
	return startInDir(dir, info)
}

func startInDir(dir string, info AppInfo) (*Session, *Report, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, nil, err
	}

	info = normalizeInfo(info)
	session := &Session{
		markerPath:  filepath.Join(dir, markerFileName),
		logPath:     filepath.Join(dir, logFileName),
		reportPath:  filepath.Join(dir, reportFileName),
		pendingPath: filepath.Join(dir, pendingFileName),
		info:        info,
		env:         currentEnvironment(),
	}
	lockFile, err := acquireLock(filepath.Join(dir, lockFileName))
	if err != nil {
		if errors.Is(err, ErrAlreadyRunning) {
			return nil, nil, err
		}
		return nil, nil, err
	}
	session.lockFile = lockFile

	if err := session.recoverStaleRun(); err != nil {
		_ = lockFile.Close()
		return nil, nil, err
	}

	logFile, err := openLog(session.logPath)
	if err != nil {
		_ = lockFile.Close()
		return nil, nil, err
	}
	session.logFile = logFile

	if err := writeJSON(session.markerPath, sessionMarker{
		Version:     info.Version,
		PID:         info.PID,
		StartedAt:   info.StartedAt,
		Environment: session.env,
	}); err != nil {
		_ = logFile.Close()
		_ = lockFile.Close()
		return nil, nil, err
	}

	pending, err := session.pendingReport()
	if err != nil {
		_ = session.logFile.Close()
		_ = lockFile.Close()
		_ = os.Remove(session.markerPath)
		return nil, nil, err
	}
	return session, pending, nil
}

func normalizeInfo(info AppInfo) AppInfo {
	if info.Version == "" {
		info.Version = "unknown"
	}
	if info.PID == 0 {
		info.PID = os.Getpid()
	}
	if info.StartedAt.IsZero() {
		info.StartedAt = time.Now().UTC()
	} else {
		info.StartedAt = info.StartedAt.UTC()
	}
	return info
}

// LogFile returns the per-user log file used by the standard logger.
func (s *Session) LogFile() *os.File {
	if s == nil {
		return nil
	}
	return s.logFile
}

// ReportPath returns the path of the latest local crash report.
func (s *Session) ReportPath() string {
	if s == nil {
		return ""
	}
	return s.reportPath
}

// RecordPanic saves a recovered panic and its stack trace for the next launch.
func (s *Session) RecordPanic(value any) {
	if s == nil {
		return
	}
	logs, _ := readTail(s.logPath, maxLogBytes)
	report := formatReportWithEnvironment(s.info, s.env, time.Now().UTC(), "panic", value, string(debug.Stack()), logs)
	_ = s.saveReport(report)
	_ = os.Remove(s.markerPath)
}

// ClearPending marks the latest report as reviewed while retaining the report
// file so the user can inspect it later.
func (s *Session) ClearPending() error {
	if s == nil {
		return nil
	}
	if err := os.Remove(s.pendingPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Close marks the process as having exited normally and closes its log file.
func (s *Session) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	var errs []error
	if err := s.logFile.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := os.Remove(s.markerPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		errs = append(errs, err)
	}
	if err := s.lockFile.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func acquireLock(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrAlreadyRunning
		}
		return nil, err
	}
	return file, nil
}

func (s *Session) recoverStaleRun() error {
	data, err := os.ReadFile(s.markerPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	marker := sessionMarker{
		Version:     s.info.Version,
		PID:         0,
		StartedAt:   s.info.StartedAt,
		Environment: s.env,
	}
	if json.Unmarshal(data, &marker) != nil {
		// A partially written marker is still evidence that the prior process
		// did not close cleanly. Keep the report useful instead of discarding it.
		marker = sessionMarker{Version: s.info.Version, StartedAt: s.info.StartedAt}
	}
	if marker.Version == "" {
		marker.Version = s.info.Version
	}
	if marker.StartedAt.IsZero() {
		marker.StartedAt = s.info.StartedAt
	}

	logs, _ := readTail(s.logPath, maxLogBytes)
	env := marker.Environment
	if env.OS == "" {
		env = s.env
	}
	report := formatReportWithEnvironment(
		AppInfo{Version: marker.Version, PID: marker.PID, StartedAt: marker.StartedAt},
		env,
		time.Now().UTC(),
		"unexpected process exit (panic or fatal signal) - no clean shutdown was recorded",
		nil,
		"",
		logs,
	)
	if err := s.saveReport(report); err != nil {
		return err
	}
	return os.Remove(s.markerPath)
}

func (s *Session) pendingReport() (*Report, error) {
	if _, err := os.Stat(s.pendingPath); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(s.reportPath)
	if errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(s.pendingPath)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Report{Text: string(data), Path: s.reportPath}, nil
}

func (s *Session) saveReport(report string) error {
	report = trim(report, maxReportBytes)
	if err := writeFile(s.reportPath, []byte(report)); err != nil {
		return err
	}
	return writeFile(s.pendingPath, []byte("pending\n"))
}

func openLog(path string) (*os.File, error) {
	if info, err := os.Stat(path); err == nil && info.Size() > maxLogBytes {
		if data, readErr := readTail(path, maxLogBytes/2); readErr == nil {
			if writeErr := os.WriteFile(path, []byte(data), 0o600); writeErr != nil {
				return nil, writeErr
			}
		}
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
}

func readTail(path string, limit int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() > limit {
		if _, err := file.Seek(-limit, io.SeekEnd); err != nil {
			return "", err
		}
	}
	data, err := io.ReadAll(io.LimitReader(file, limit))
	return string(data), err
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFile(path, data)
}

func writeFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".lamha-crash-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func stateDir() (string, error) {
	if base := os.Getenv("XDG_STATE_HOME"); filepath.IsAbs(base) {
		return filepath.Join(base, "lamha"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "lamha"), nil
}

type environment struct {
	OS            string
	Architecture  string
	Go            string
	Desktop       string
	Session       string
	Backend       string
	DisplayServer string
	Locale        string
	Distribution  string
	Installation  string
}

func currentEnvironment() environment {
	return environment{
		OS:            runtime.GOOS,
		Architecture:  runtime.GOARCH,
		Go:            runtime.Version(),
		Desktop:       envOrUnknown("XDG_CURRENT_DESKTOP"),
		Session:       envOrUnknown("XDG_SESSION_TYPE"),
		Backend:       envOrUnknown("GDK_BACKEND"),
		DisplayServer: displayServer(),
		Locale:        envLocale(),
		Distribution:  distribution(),
		Installation:  installation(),
	}
}

func envOrUnknown(name string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return "unknown"
}

func envLocale() string {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return "unknown"
}

func displayServer() string {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "Wayland"
	}
	if os.Getenv("DISPLAY") != "" {
		return "X11"
	}
	return "unknown"
}

func distribution() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		}
	}
	return "unknown"
}

func installation() string {
	if os.Getenv("APPIMAGE") != "" {
		return "AppImage"
	}
	return "native binary or source"
}

var (
	homePathPattern  = regexp.MustCompile(`/home/[^/[:space:]]+`)
	mountPathPattern = regexp.MustCompile(`/tmp/\.mount_[^/[:space:]]+`)
	waylandIDPattern = regexp.MustCompile(`(?i)(?:wayland|x11):[A-Za-z0-9._:/-]+`)
)

func sanitizeLogs(logs string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		logs = strings.ReplaceAll(logs, home, "<HOME>")
	}
	logs = homePathPattern.ReplaceAllString(logs, "<HOME>")
	logs = mountPathPattern.ReplaceAllString(logs, "<APPIMAGE-MOUNT>")
	logs = waylandIDPattern.ReplaceAllString(logs, "<WINDOW-HANDLE>")
	return logs
}

func formatReport(info AppInfo, occurredAt time.Time, reason string, panicValue any, stack, logs string) string {
	return formatReportWithEnvironment(info, currentEnvironment(), occurredAt, reason, panicValue, stack, logs)
}

func formatReportWithEnvironment(info AppInfo, env environment, occurredAt time.Time, reason string, panicValue any, stack, logs string) string {
	var b strings.Builder
	b.WriteString("Lamha crash report\n")
	fmt.Fprintf(&b, "Generated at: %s\n", occurredAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "Reason: %s\n", reason)
	fmt.Fprintf(&b, "Lamha version: %s\n", info.Version)
	fmt.Fprintf(&b, "Process ID: %d\n", info.PID)
	fmt.Fprintf(&b, "Session started: %s\n", info.StartedAt.UTC().Format(time.RFC3339))
	b.WriteString("\nEnvironment:\n")
	fmt.Fprintf(&b, "- Operating system: %s\n", env.OS)
	fmt.Fprintf(&b, "- Distribution: %s\n", env.Distribution)
	fmt.Fprintf(&b, "- Architecture: %s\n", env.Architecture)
	fmt.Fprintf(&b, "- Go runtime: %s\n", env.Go)
	fmt.Fprintf(&b, "- Desktop: %s\n", env.Desktop)
	fmt.Fprintf(&b, "- Session type: %s\n", env.Session)
	fmt.Fprintf(&b, "- Display server: %s\n", env.DisplayServer)
	fmt.Fprintf(&b, "- Graphics backend: %s\n", env.Backend)
	fmt.Fprintf(&b, "- Locale: %s\n", env.Locale)
	fmt.Fprintf(&b, "- Installation: %s\n", env.Installation)

	if panicValue != nil {
		b.WriteString("\nPanic value:\n")
		fmt.Fprintf(&b, "%v\n", panicValue)
		b.WriteString("\nStack trace:\n")
		b.WriteString(stack)
		if !strings.HasSuffix(stack, "\n") {
			b.WriteByte('\n')
		}
	}

	b.WriteString("\nRecent log output (sanitized):\n")
	if logs == "" {
		b.WriteString("(no log output was available)\n")
	} else {
		b.WriteString(sanitizeLogs(logs))
		if !strings.HasSuffix(logs, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// IssueURL builds an explicit, pre-filled GitHub issue URL. The report is
// truncated for URL-size limits; the complete report remains on disk.
func IssueURL(repoURL string, report Report) string {
	base := strings.TrimRight(repoURL, "/") + "/new"
	u, err := url.Parse(base)
	if err != nil {
		return ""
	}
	body := "## What happened?\n\nLamha stopped unexpectedly. Please describe what you were doing before the crash.\n\n" +
		"## Diagnostics\n\n```text\n" + trim(strings.ReplaceAll(report.Text, "```", "'''"), maxIssueReportBytes) + "\n```\n\n" +
		"## Reproduction steps\n\n1.\n2.\n3.\n"
	query := u.Query()
	query.Set("title", "[Crash] Lamha stopped unexpectedly")
	query.Set("body", body)
	u.RawQuery = query.Encode()
	return u.String()
}

func trim(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "\n[report truncated; use Copy report or attach the local file]\n"
}

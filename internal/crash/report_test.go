package crash

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testInfo() AppInfo {
	return AppInfo{
		Version:   "26.09.0-test",
		PID:       4242,
		StartedAt: time.Date(2026, 9, 2, 10, 11, 12, 0, time.UTC),
	}
}

func TestStartAndCloseRemoveActiveMarker(t *testing.T) {
	dir := t.TempDir()
	session, pending, err := startInDir(dir, testInfo())
	if err != nil {
		t.Fatal(err)
	}
	if pending != nil {
		t.Fatal("fresh session unexpectedly has a pending report")
	}
	if _, err := os.Stat(filepath.Join(dir, markerFileName)); err != nil {
		t.Fatalf("active marker missing: %v", err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, markerFileName)); !os.IsNotExist(err) {
		t.Fatalf("active marker still exists: %v", err)
	}
}

func TestStaleSessionBecomesPendingReport(t *testing.T) {
	dir := t.TempDir()
	first, _, err := startInDir(dir, testInfo())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.LogFile().WriteString("capture failed for /home/alice/Pictures/shot.png\n"); err != nil {
		t.Fatal(err)
	}
	// Simulate SIGKILL: the marker is left behind, but the log descriptor is
	// closed so the next process can open the same file cleanly.
	if err := first.LogFile().Close(); err != nil {
		t.Fatal(err)
	}
	if err := first.lockFile.Close(); err != nil {
		t.Fatal(err)
	}

	second, pending, err := startInDir(dir, AppInfo{Version: "26.09.1-test"})
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if pending == nil {
		t.Fatal("stale session did not create a pending report")
	}
	if !strings.Contains(pending.Text, "unexpected process exit") {
		t.Fatal("pending report did not explain the unexpected exit")
	}
	if !strings.Contains(pending.Text, "capture failed") {
		t.Fatal("pending report omitted recent logs")
	}
	if !strings.Contains(pending.Text, "<HOME>") {
		t.Fatal("pending report did not sanitize a home path")
	}
	if _, err := os.Stat(filepath.Join(dir, markerFileName)); err != nil {
		t.Fatalf("new session marker missing: %v", err)
	}
}

func TestSecondSessionDoesNotOverwritePrimaryMarker(t *testing.T) {
	dir := t.TempDir()
	first, _, err := startInDir(dir, testInfo())
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if _, _, err := startInDir(dir, testInfo()); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second session error = %v, want ErrAlreadyRunning", err)
	}
}

func TestRecordPanicWritesStackAndPendingReport(t *testing.T) {
	dir := t.TempDir()
	session, _, err := startInDir(dir, testInfo())
	if err != nil {
		t.Fatal(err)
	}
	session.RecordPanic("boom")
	data, err := os.ReadFile(filepath.Join(dir, reportFileName))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "Reason: panic") || !strings.Contains(text, "boom") {
		t.Fatalf("panic report missing reason/value:\n%s", text)
	}
	if !strings.Contains(text, "Stack trace:") {
		t.Fatal("panic report missing stack trace")
	}
	if _, err := os.Stat(filepath.Join(dir, pendingFileName)); err != nil {
		t.Fatalf("pending marker missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, markerFileName)); !os.IsNotExist(err) {
		t.Fatalf("panic left active marker: %v", err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueURLIncludesRequiredReportSections(t *testing.T) {
	report := Report{Text: "Lamha version: 26.09.0\nReason: panic\nStack trace: boom"}
	issue := IssueURL("https://github.com/Zyzto/Lamha/issues", report)
	parsed, err := url.Parse(issue)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "/Zyzto/Lamha/issues/new" {
		t.Fatalf("issue path = %q", parsed.Path)
	}
	query := parsed.Query()
	if query.Get("title") == "" || !strings.Contains(query.Get("title"), "Crash") {
		t.Fatalf("issue title = %q", query.Get("title"))
	}
	body := query.Get("body")
	for _, want := range []string{"What happened?", "Diagnostics", "Reproduction steps", "Reason: panic"} {
		if !strings.Contains(body, want) {
			t.Fatalf("issue body missing %q:\n%s", want, body)
		}
	}
}

func TestIssueURLTruncatesLargeReport(t *testing.T) {
	report := Report{Text: strings.Repeat("x", maxIssueReportBytes*2)}
	issue := IssueURL("https://github.com/Zyzto/Lamha/issues", report)
	parsed, err := url.Parse(issue)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(parsed.Query().Get("body")); got > maxIssueReportBytes+512 {
		t.Fatalf("issue body too large: %d bytes", got)
	}
	if !strings.Contains(parsed.Query().Get("body"), "report truncated") {
		t.Fatal("issue body did not explain truncation")
	}
}

func TestClearPendingKeepsReport(t *testing.T) {
	dir := t.TempDir()
	session, _, err := startInDir(dir, testInfo())
	if err != nil {
		t.Fatal(err)
	}
	session.RecordPanic("boom")
	if err := session.ClearPending(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, pendingFileName)); !os.IsNotExist(err) {
		t.Fatalf("pending marker still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, reportFileName)); err != nil {
		t.Fatalf("report was removed with pending marker: %v", err)
	}
	session.Close()
}

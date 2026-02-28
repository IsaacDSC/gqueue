package main

import (
	"bufio"
	"bytes"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer wraps a buffer so stdout and stderr can be written concurrently without dropping output.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// buildApiBinary builds ./cmd/api and returns the path to the binary.
func buildApiBinary(t *testing.T, moduleRoot string) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "gqueue-api")
	if os.PathListSeparator == ';' {
		exe += ".exe"
	}
	build := exec.Command("go", "build", "-o", exe, "./cmd/api")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/api: %v\n%s", err, out)
	}
	return exe
}

// loadEnvFile reads path and sets env vars from KEY=VALUE lines (comments and empty lines ignored).
func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(line[7:])
		}
		i := strings.Index(line, "=")
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		value := strings.TrimSpace(line[i+1:])
		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
		_ = os.Setenv(key, value)
	}
}

// holdBackofficeConnection opens a TCP connection to the backoffice server and sends an
// incomplete HTTP request so the server blocks reading until ReadTimeout. When we SIGINT,
// the server must wait for this connection to drain before exiting—proving shutdown waits.
func holdBackofficeConnection(t *testing.T) (closeConn func()) {
	t.Helper()
	port := os.Getenv("BACKOFFICE_API_PORT")
	if port == "" {
		port = "8081"
	}
	addr := net.JoinHostPort("localhost", port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Skipf("Cannot connect to backoffice at %s (server may not be up): %v", addr, err)
		return func() {}
	}
	// Incomplete request: no \r\n\r\n so server keeps reading (until ReadTimeout).
	_, _ = conn.Write([]byte("GET /health HTTP/1.1\r\nHost: localhost\r\n"))
	return func() { _ = conn.Close() }
}

// requiredShutdownLogs are the log messages that must appear (see main.go waitForShutdown).
// We require the first and last so we know shutdown started and completed; the duration lines
// in between may be interleaved with asynq logs and can be flaky to capture.
const minShutdownDuration = 2 * time.Second

var requiredShutdownLogs = []string{
	"Shutting down servers...",
	"All servers shutdown complete",
}

func TestShutdownGraceful(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Find module root (same dir as go.mod).
	moduleRoot := wd
	for {
		goMod := filepath.Join(moduleRoot, "go.mod")
		if _, err := os.Stat(goMod); err == nil {
			break
		}
		parent := filepath.Dir(moduleRoot)
		if parent == moduleRoot {
			t.Fatalf("go.mod not found (searched from %s)", wd)
		}
		moduleRoot = parent
	}

	// Load .env from module root so "go test" can use local env (only stdlib, no godotenv).
	loadEnvFile(filepath.Join(moduleRoot, ".env"))

	// Skip if required env is missing (integration test needs Redis + DB).
	if os.Getenv("DB_CONNECTION_STRING") == "" || os.Getenv("CACHE_ADDR") == "" {
		t.Skip("Skipping shutdown integration test: DB_CONNECTION_STRING and CACHE_ADDR must be set (use .env or run scripts/run_shutdown_test.sh)")
	}

	// Build and run the binary so SIGINT goes to the server process (go run would send it to the go process only).
	exe := buildApiBinary(t, moduleRoot)
	cmd := exec.Command(exe)
	cmd.Dir = moduleRoot
	output := &syncBuffer{}
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// Wait for server to be up (main finishes init and enters waitForShutdown).
	time.Sleep(3 * time.Second)

	if cmd.Process == nil {
		t.Fatalf("cmd.Process is nil after Start")
	}

	// Hold one connection with an incomplete HTTP request so the server must wait for it
	// (or ReadTimeout) during shutdown. This proves shutdown is not killing immediately.
	closeConn := holdBackofficeConnection(t)
	defer closeConn()

	// Give the server time to see the connection (in-flight).
	time.Sleep(500 * time.Millisecond)

	startShutdown := time.Now()
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("Failed to send SIGINT: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case waitErr := <-done:
		shutdownDuration := time.Since(startShutdown)
		outStr := output.String()

		// (1) Process must exit with code 0 (graceful exit). Non-zero = panic, crash, or unclean exit.
		if waitErr != nil {
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				t.Errorf("Process did not exit gracefully: exit code %d. Output:\n%s", exitErr.ExitCode(), outStr)
			} else {
				t.Errorf("Process wait error: %v", waitErr)
			}
		}

		// (2) All shutdown log lines from waitForShutdown must appear.
		for _, want := range requiredShutdownLogs {
			if !strings.Contains(outStr, want) {
				t.Errorf("Shutdown log missing: %q. Full output:\n%s", want, outStr)
			}
		}

		t.Logf("Shutdown completed in %v", shutdownDuration)

		// (3) Shutdown must take at least minShutdownDuration: we held an in-flight connection
		// (incomplete HTTP request); the server must wait for it before exiting.
		if shutdownDuration < minShutdownDuration {
			t.Errorf("Shutdown did not wait for in-flight connection: took %v, expected at least %v", shutdownDuration, minShutdownDuration)
		}
		// (4) Shutdown should complete under the 15s select timeout (server uses 1m timeout internally).
		if shutdownDuration > 14*time.Second {
			t.Errorf("Shutdown took too long: %v", shutdownDuration)
		}
	case <-time.After(15 * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		t.Fatalf("Shutdown did not complete within 15s. Output so far:\n%s", output.String())
	}
}

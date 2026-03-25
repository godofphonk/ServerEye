// Package executor provides a resource-safe wrapper around os/exec that enforces
// per-command timeouts, a global concurrency limit, and a per-command stdout cap.
// These controls collectively prevent the Denial-of-Service scenarios that arise
// when metrics collection commands (nvidia-smi, sensors, etc.) hang indefinitely,
// run in unbounded parallel, or produce arbitrarily large output.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

const (
	// DefaultMaxConcurrency is the default maximum number of commands that may run
	// in parallel.  Keeping this low prevents thread/FD exhaustion.
	DefaultMaxConcurrency = 4

	// DefaultMaxOutputBytes is the default cap on a command's stdout (10 MiB).
	// Output beyond this limit is silently discarded to prevent memory exhaustion.
	DefaultMaxOutputBytes = 10 * 1024 * 1024
)

// CommandExecutor runs system commands with:
//   - a semaphore-based concurrency limit (prevents resource exhaustion),
//   - a per-command timeout via exec.CommandContext (prevents hangs and zombie accumulation),
//   - a bounded stdout buffer (prevents memory exhaustion from large outputs).
type CommandExecutor struct {
	sem    chan struct{}
	maxOut int64
}

// New creates a CommandExecutor.
// Non-positive maxConcurrency uses DefaultMaxConcurrency.
// Non-positive maxOutputBytes uses DefaultMaxOutputBytes.
func New(maxConcurrency int, maxOutputBytes int64) *CommandExecutor {
	if maxConcurrency <= 0 {
		maxConcurrency = DefaultMaxConcurrency
	}
	if maxOutputBytes <= 0 {
		maxOutputBytes = DefaultMaxOutputBytes
	}
	return &CommandExecutor{
		sem:    make(chan struct{}, maxConcurrency),
		maxOut: maxOutputBytes,
	}
}

// Run executes name with args, blocking until the command finishes, the timeout
// elapses, or ctx is cancelled — whichever comes first.
//
// Concurrency: at most maxConcurrency commands run simultaneously; excess callers
// block until a slot is available or ctx is cancelled.
//
// Timeout: the command is killed (SIGKILL via exec.CommandContext) when timeout
// expires, preventing zombie accumulation — Go's runtime always calls Wait() to
// reap the child even after the kill.
//
// Output: stdout is captured up to maxOutputBytes; any excess is silently dropped.
// stderr is not included in the returned bytes.
func (e *CommandExecutor) Run(ctx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error) {
	// Acquire a concurrency slot; honour caller cancellation while waiting.
	select {
	case e.sem <- struct{}{}:
		defer func() { <-e.sem }()
	case <-ctx.Done():
		return nil, fmt.Errorf("executor: context cancelled before slot available: %w", ctx.Err())
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, name, args...)

	var buf bytes.Buffer
	cmd.Stdout = &limitWriter{w: &buf, limit: e.maxOut}
	// Discard stderr to avoid it mixing into the returned output.

	if err := cmd.Run(); err != nil {
		if cmdCtx.Err() != nil {
			return nil, fmt.Errorf("command %q timed out after %s", name, timeout)
		}
		return nil, err
	}

	return buf.Bytes(), nil
}

// limitWriter is an io.Writer that silently discards bytes written beyond limit.
// It always reports the full len(p) as consumed so that callers (the os/exec
// output plumbing) never see a short-write error.
type limitWriter struct {
	w     io.Writer
	limit int64
	n     int64
}

func (lw *limitWriter) Write(p []byte) (int, error) {
	if lw.n >= lw.limit {
		// Already at/over the limit; silently discard.
		return len(p), nil
	}
	remaining := lw.limit - lw.n
	toWrite := p
	if int64(len(p)) > remaining {
		toWrite = p[:remaining]
	}
	n, err := lw.w.Write(toWrite)
	lw.n += int64(n)
	if err != nil {
		return n, err
	}
	// Report the original len(p) so callers see a successful full write.
	return len(p), nil
}

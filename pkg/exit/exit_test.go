package exit

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fatalSubprocessEnv marks the re-executed child that is supposed to die.
const fatalSubprocessEnv = "BE_FATAL"

// fakeTerminal forces the terminal check, which is always false under `go test`.
func fakeTerminal(t *testing.T, interactive bool) {
	t.Helper()

	original := isTerminal
	isTerminal = func(int) bool { return interactive }

	t.Cleanup(func() { isTerminal = original })
}

// feedStdin replaces stdin with a pipe holding input. MakeRaw fails on a pipe, so
// this also exercises the read-a-line fallback.
func feedStdin(t *testing.T, input string) {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	_, err = writer.WriteString(input)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	original := stdin
	stdin = reader

	t.Cleanup(func() {
		stdin = original

		_ = reader.Close()
	})
}

// capturePrompt redirects the prompt so a test can assert on what was written.
func capturePrompt(t *testing.T) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	original := promptWriter
	promptWriter = buf

	t.Cleanup(func() { promptWriter = original })

	return buf
}

func TestPauseForKeyWaitsOnTerminal(t *testing.T) { //nolint:paralleltest // swaps package-level stdin and terminal hooks.
	fakeTerminal(t, true)
	feedStdin(t, "x\n")

	buf := capturePrompt(t)

	PauseForKey()

	assert.Contains(t, buf.String(), "Press any key to continue")
}

func TestPauseForKeySkipsWhenStdinIsNotATerminal(t *testing.T) { //nolint:paralleltest // swaps package-level stdin and terminal hooks.
	fakeTerminal(t, false)
	feedStdin(t, "")

	buf := capturePrompt(t)

	PauseForKey()

	assert.Empty(t, buf.String(), "a non-interactive run must not prompt")
}

func TestPauseForKeyHonoursNoPauseEnv(t *testing.T) { //nolint:paralleltest // sets an env var and swaps package-level hooks.
	t.Setenv(NoPauseEnv, "1")
	fakeTerminal(t, true)
	feedStdin(t, "")

	buf := capturePrompt(t)

	PauseForKey()

	assert.Empty(t, buf.String(), "NoPauseEnv must suppress the prompt even on a terminal")
}

// TestFatalfLogsAndExits re-execs this test in a child process, because Fatalf
// terminates whatever process calls it.
func TestFatalfLogsAndExits(t *testing.T) { //nolint:paralleltest // re-execs the test binary and reads its exit status.
	if os.Getenv(fatalSubprocessEnv) == "1" {
		Fatalf("startup failed: %s", "bad config")

		return
	}

	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestFatalfLogsAndExits") // #nosec G204 -- re-execs this test binary.
	cmd.Env = append(os.Environ(), fatalSubprocessEnv+"=1")

	output, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError

	require.ErrorAs(t, err, &exitErr, "Fatalf must exit non-zero")
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Contains(t, string(output), "startup failed: bad config")
	assert.NotContains(t, string(output), "Press any key", "the child has no terminal, so it must not pause")
}

// TestReadKeyFallsBackToLine documents that a stdin that cannot be put in raw
// mode still blocks until input arrives, rather than being skipped.
func TestReadKeyFallsBackToLine(t *testing.T) { //nolint:paralleltest // swaps package-level stdin.
	feedStdin(t, "abc\nleftover")

	readKey(int(stdin.Fd()))

	rest, err := io.ReadAll(stdin)
	require.True(t, err == nil || errors.Is(err, os.ErrClosed))
	assert.False(t, strings.Contains(string(rest), "abc"), "the first line should have been consumed")
}

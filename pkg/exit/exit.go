// Package exit terminates Console after giving the operator a chance to read why.
//
// A double-clicked binary owns its console window, and the window dies with the
// process — on Windows immediately, before anything printed by log.Fatalf can be
// read. Fatal paths therefore wait for a keystroke first, but only when a terminal
// is actually attached, so containers, services, and CI keep exiting immediately.
package exit

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/term"
)

// NoPauseEnv suppresses the pause. Set it for runners that attach a TTY
// (`docker run -it`, some supervisors) but have nobody present to press a key.
const NoPauseEnv = "DMT_NO_PAUSE_ON_EXIT"

// Prompt is what the operator sees while Console waits.
const Prompt = "\nPress any key to continue ..."

// Indirected for tests: a test binary has no controlling terminal, so the pause
// would otherwise never be exercised.
var (
	promptWriter io.Writer = os.Stderr // stderr, where the fatal message also went
	stdin                  = os.Stdin
	isTerminal             = term.IsTerminal
)

// Fatalf logs like log.Fatalf, waits for a keystroke, then exits with status 1.
// It never returns.
func Fatalf(format string, v ...any) {
	log.Printf(format, v...)
	PauseForKey()
	os.Exit(1)
}

// PauseForKey blocks until the operator presses a key. It returns at once when
// NoPauseEnv is set or when stdin is not an interactive terminal — a pipe,
// /dev/null, or a redirected file — because then no keystroke is ever coming.
func PauseForKey() {
	if os.Getenv(NoPauseEnv) != "" {
		return
	}

	fd := int(stdin.Fd())
	if !isTerminal(fd) {
		return
	}

	fmt.Fprint(promptWriter, Prompt)
	readKey(fd)
	fmt.Fprintln(promptWriter)
}

// readKey consumes a single keystroke. Raw mode is what makes "any key" literal:
// a cooked terminal releases input only at end of line, so without it the prompt
// would really mean "press Enter". A terminal that refuses raw mode degrades to
// waiting for a line, which is still better than not waiting at all.
func readKey(fd int) {
	state, err := term.MakeRaw(fd)
	if err != nil {
		_, _ = bufio.NewReader(stdin).ReadString('\n')

		return
	}

	defer func() {
		_ = term.Restore(fd, state)
	}()

	var key [1]byte

	_, _ = stdin.Read(key[:])
}

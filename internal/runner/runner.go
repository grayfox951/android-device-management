// Package runner executes external commands and captures their output.
package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Result is the outcome of a single command invocation.
type Result struct {
	// Command is the command line as it was run, for display purposes.
	Command string
	// Out holds the combined stdout and stderr.
	Out string
	// Err is a non-nil when the process could not be started at all.
	Err error
	// Code is the process exit status; -1 when the process never ran.
	Code int
	// Duration is the wall time the command took.
	Duration time.Duration
}

// OK reports whether the command finished with exit status zero.
func (r Result) OK() bool { return r.Err == nil && r.Code == 0 }

// Trimmed returns the output without surrounding whitespace.
func (r Result) Trimmed() string { return strings.TrimSpace(r.Out) }

// Empty reports whether the command produced no output at all.
func (r Result) Empty() bool { return strings.TrimSpace(r.Out) == "" }

// shellQuote renders an argument list the way a shell would accept it.
func shellQuote(args []string) string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\n\"'\\$`") {
			out = append(out, fmt.Sprintf("%q", a))
			continue
		}
		out = append(out, a)
	}
	return strings.Join(out, " ")
}

// Run executes a command and returns all of its output. The context is used
// for cancellation; a deadline should be set by the caller for commands that
// may hang, such as anything that waits on a missing device.
func Run(ctx context.Context, name string, args ...string) Result {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(cmd.Environ(), "LC_ALL=C", "LANG=C")

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	start := time.Now()
	err := cmd.Run()
	res := Result{
		Command:  shellQuote(append([]string{name}, args...)),
		Out:      buf.String(),
		Duration: time.Since(start),
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			res.Code = ee.ExitCode()
		} else {
			// The process never ran: exec could not find the binary or the
			// context was cancelled before the start.
			res.Err = err
			res.Code = -1
		}
	}
	return res
}

// RunQuiet is Run for the many adb queries where only success matters. It
// applies a short timeout so a hung daemon cannot freeze the interface.
func RunQuiet(ctx context.Context, name string, args ...string) Result {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
	}
	return Run(ctx, name, args...)
}

// RunShell parses a command line and runs the resulting argv. It is used for
// the manual install fallback where the user types a whole shell command.
func RunShell(ctx context.Context, line string) Result {
	argv, err := Split(line)
	if err != nil {
		return Result{Command: line, Err: err, Code: -1}
	}
	if len(argv) == 0 {
		return Result{Command: line, Err: fmt.Errorf("empty command"), Code: -1}
	}
	return Run(ctx, argv[0], argv[1:]...)
}

// Split tokenises a command line honouring quotes and backslash escapes.
func Split(line string) ([]string, error) {
	var (
		args    []string
		cur     strings.Builder
		quote   rune
		escaped bool
		started bool
	)
	flush := func() {
		if started {
			args = append(args, cur.String())
			cur.Reset()
			started = false
		}
	}
	for _, r := range line {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
			started = true
		case r == '\\' && quote != '\'':
			escaped = true
			started = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			started = true
		case r == ' ' || r == '\t' || r == '\n':
			flush()
		default:
			cur.WriteRune(r)
			started = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unbalanced quote")
	}
	flush()
	return args, nil
}

// ErrExit reports a command that ran but returned a non-zero status.
type ErrExit struct {
	Command string
	Code    int
}

func (e ErrExit) Error() string {
	return fmt.Sprintf("%s exited with code %d", e.Command, e.Code)
}

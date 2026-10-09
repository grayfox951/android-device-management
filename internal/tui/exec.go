package tui

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"adm/internal/android"
	"adm/internal/i18n"
)

// guardTimeout bounds a Guard, which is normally a single `cat`.
// tickInterval is the polling period for a running command.
// guardDoneMsg carries the verdict of a Guard back to the model.
type guardDoneMsg struct {
	cmd  android.Command
	vals map[string]string
	ok   bool
	key  string
	args []any
}

// runGuard executes a command's guard in the background.
func runGuard(c android.Command, d android.Device, vals map[string]string) tea.Cmd {
	// The device is copied into the closure so the guard sees the same
	// selection the user is looking at.
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), guardTimeout)
		defer cancel()

		ok, key, args := c.Guard(ctx, d, vals)
		return guardDoneMsg{cmd: c, vals: vals, ok: ok, key: key, args: args}
	}
}

// handleGuard applies a guard verdict: either the command starts, or the
// refusal is shown and nothing runs.
func (m *Model) handleGuard(msg guardDoneMsg) (tea.Model, tea.Cmd) {
	if !msg.ok {
		m.blocked = i18nT(msg.key, msg.args...)
		return m, nil
	}
	return m.runNow(msg.cmd, msg.vals)
}

// runNow starts the process for a command that has passed every check.
func (m *Model) runNow(c android.Command, vals map[string]string) (tea.Model, tea.Cmd) {
	argv := c.Build(m.device, vals)
	if len(argv) == 0 {
		m.notice = i18nT("err.exec", "empty argv")
		return m, nil
	}

	// Clean up the previous log before starting a new one.
	if m.run.logPath != "" && m.run.savedTo == "" {
		_ = os.Remove(m.run.logPath)
	}

	logFile, err := os.CreateTemp("", "adm-log-*.txt")
	if err != nil {
		m.err = err
		return m, nil
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C", "TERM=dumb")

	// Commands that dump binary or huge output write it straight to the file
	// the user picked, keeping the screen readable.
	var redirectPath string
	var out io.Writer = logFile
	if c.Redirect != "" {
		if p := vals[c.Redirect]; p != "" {
			target := p
			rf, err := os.Create(target)
			if err != nil {
				_ = logFile.Close()
				m.err = err
				return m, nil
			}
			defer rf.Close()
			out = rf
			redirectPath = target
		}
	}
	cmd.Stdout = out
	cmd.Stderr = logFile

	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		_ = os.Remove(logFile.Name())
		m.err = err
		return m, nil
	}

	go func() {
		runErr := cmd.Wait()
		_ = logFile.Close()
		done <- runErr
	}()

	r := &m.run
	r.title = c.Label
	r.cmdLine = strings.Join(argv, " ")
	r.logPath = logFile.Name()
	r.running = true
	r.done = false
	r.code = -1
	r.runErr = nil
	r.started = time.Now()
	r.redirect = redirectPath
	r.savedTo = ""
	r.doneCh = done
	r.lines = nil
	r.total = 0
	r.partial = ""
	r.offset = 0

	m.stack = append(m.stack, screenCommands)
	m.screen = screenOutput
	m.notice = ""
	r.resize(m.w, m.h)
	return m, r.tick()
}

// tick schedules the next poll of a running command.
func (r *runState) tick() tea.Cmd {
	if !r.running {
		return nil
	}
	return tea.Tick(tickInterval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

// poll reads whatever the process wrote since the last check and picks up the
// exit status once the process is gone.
func (r *runState) poll() {
	if !r.running {
		return
	}
	select {
	case err := <-r.doneCh:
		// Read before recording the status: a command that finished between
		// two ticks has everything still sitting unread in the log.
		r.drain()
		r.flushPartial()
		r.finish(err)
		r.running = false
		r.appendStatus()
	default:
		r.drain()
	}
}

// drain appends newly written bytes from the log to the in-memory lines.
func (r *runState) drain() {
	if r.logPath == "" {
		return
	}
	f, err := os.Open(r.logPath)
	if err != nil {
		return
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil || st.Size() <= r.offset {
		return
	}
	if _, err := f.Seek(r.offset, io.SeekStart); err != nil {
		return
	}

	buf := make([]byte, st.Size()-r.offset)
	n, _ := io.ReadFull(f, buf)
	r.offset += int64(n)

	data := r.partial + string(buf)
	lines := strings.Split(data, "\n")
	r.partial = lines[len(lines)-1]

	for _, l := range lines[:len(lines)-1] {
		r.append(strings.TrimRight(l, "\r"))
	}
}

// flushPartial turns a trailing fragment without a newline into a line.
func (r *runState) flushPartial() {
	if r.partial == "" {
		return
	}
	r.append(r.partial)
	r.partial = ""
}

func (r *runState) append(line string) {
	r.lines = append(r.lines, line)
	r.total++
	if len(r.lines) > maxLines {
		r.lines = r.lines[len(r.lines)-maxLines:]
	}
}

// finish records the exit status of the command.
func (r *runState) finish(err error) {
	r.done = true
	r.took = time.Since(r.started)

	if err == nil {
		r.code = 0
		return
	}
	r.runErr = err
	if ee, ok := err.(*exec.ExitError); ok {
		r.code = ee.ExitCode()
		return
	}
	r.code = -1
}

func (r *runState) appendStatus() {
	switch {
	case r.runErr != nil && r.code < 0:
		r.append(i18nT("err.exec", r.runErr))
	case r.code == 0:
		r.append("✓ " + i18nT("common.done") + " · " + formatDuration(r.took))
	default:
		r.append("✗ " + i18nT("common.failed") + " · " + formatDuration(r.took) +
			" · " + i18nT("common.exitcode", r.code))
	}
	if r.redirect != "" {
		r.append("→ " + r.redirect)
	}
}

// refreshViewport pushes the current lines into the scrollable widget.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return "<1ms"
	}
	if d < time.Second {
		return itoa(int(d/time.Millisecond)) + "ms"
	}
	return itoa(int(d/time.Second)) + "." + itoa(int(d%time.Second*10)/100000000) + "s"
}

// saveLog copies the whole log next to the user's home directory.
func (m *Model) saveLog() tea.Cmd {
	// First make sure the display shows everything that was written.
	m.run.drain()
	m.run.refreshViewport()

	path := m.run.logPath
	if path == "" {
		return nil
	}

	dir := homeDir()
	name := "adm-log-" + safeSerial(m.device.Serial) + "-" + time.Now().Format("20060102-150405") + ".txt"
	target := dir + "/" + name

	return func() tea.Msg {
		if err := copyFile(path, target); err != nil {
			return saveMsg{err: err}
		}
		return saveMsg{path: target}
	}
}

type saveMsg struct {
	path string
	err  error
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return "."
	}
	return h
}

func safeSerial(s string) string {
	if s == "" {
		s = "device"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// i18nT formats a translated string, kept short because the execution code
// mentions translated text on nearly every branch.
func i18nT(key string, args ...any) string {
	return i18n.T(key, args...)
}

// fullArgs merges the collected values with the command defaults.
func (m Model) fullArgs(c android.Command) map[string]string {
	full := map[string]string{}
	for _, a := range c.Args {
		full[a.Key] = a.Default
	}
	for k, v := range m.argVals {
		full[k] = v
	}
	return full
}

// missingArg returns the first argument that is still empty, if any.
func missingArg(c android.Command, vals map[string]string) string {
	for _, a := range c.Args {
		if vals[a.Key] == "" {
			return a.Label
		}
	}
	return ""
}

// launch starts a command and switches to the output screen.
func (m *Model) launch(c android.Command) (tea.Model, tea.Cmd) {
	vals := m.fullArgs(c)
	if missing := missingArg(c, vals); missing != "" {
		m.notice = i18nT("err.miss.arg", missing)
		m.screen = screenArgs
		m.argIdx = 0
		if !c.NeedsArgs() {
			m.screen = screenCommands
		}
		return m, nil
	}

	// A guard may refuse the command, and inspecting an argument can take
	// time, so it runs outside the update loop.
	if c.Guard != nil {
		return m, runGuard(c, m.device, vals)
	}
	return m.runNow(c, vals)
}

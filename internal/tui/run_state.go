package tui

import (
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"

	"adm/internal/i18n"
)

// runState is the live state of one command execution.
type runState struct {
	title    string
	cmdLine  string
	logPath  string
	offset   int64
	partial  string
	lines    []string
	total    int
	running  bool
	done     bool
	code     int
	runErr   error
	started  time.Time
	took     time.Duration
	redirect string
	savedTo  string

	doneCh chan error
	vp     viewport.Model
	w, h   int
}

// reset clears the state and makes room for a new run.
func (r *runState) reset(w, h int) {
	// The previous log is removed unless the user saved it.
	if r.logPath != "" && r.savedTo == "" {
		_ = os.Remove(r.logPath)
	}
	*r = runState{w: r.w, h: r.h}
	if r.vp.Width == 0 {
		r.resize(w, h)
	}
}

func (r *runState) resize(w, h int) {
	r.w, r.h = w, h
	vh := h - 12
	if vh < 3 {
		vh = 3
	}
	// Leave room for the box border, its padding and a small margin.
	vw := w - 10
	if vw < 20 {
		vw = 20
	}
	if r.vp.Width == 0 {
		r.vp = viewport.New(vw, vh)
	} else {
		r.vp.Width = vw
		r.vp.Height = vh
	}
	r.refreshViewport()
}

func (r *runState) refreshViewport() {
	follow := r.vp.AtBottom()

	text := strings.Join(r.lines, "\n")
	if r.partial != "" {
		text += "\n" + r.partial
	}
	if text == "" {
		text = i18n.T("err.empty.output")
	}

	r.vp.SetContent(text)
	if follow {
		r.vp.GotoBottom()
	}
}

package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"adm/internal/android"
	"adm/internal/i18n"
	"adm/internal/sysinfo"
)

func newFileBrowser(kind android.ArgKind, current string) *fileBrowser {
	fb := &fileBrowser{kind: kind, hidden: false, viewW: 110}
	fb.dir = startingDir(current)
	fb.reload(0)
	return fb
}

// startingDir decides where the browser opens: the user's home by default,
// the current value's folder when one is already set, and / as a last resort.
func startingDir(current string) string {
	home := sysinfo.Home()
	if current == "" {
		return home
	}
	abs, err := filepath.Abs(strings.TrimSpace(current))
	if err != nil {
		return home
	}
	abs = filepath.Clean(abs)

	if fi, err := os.Stat(abs); err == nil {
		if fi.IsDir() {
			return abs
		}
		return filepath.Dir(abs)
	}
	// The value may be a partial name, so fall back to its parent.
	if d := filepath.Dir(abs); d != abs {
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			return d
		}
	}
	return home
}

// reload re-reads the current directory, keeping the cursor where it can.
func (fb *fileBrowser) reload(keep int) {
	fb.entries = nil
	fb.err = nil

	items, err := os.ReadDir(fb.dir)
	if err != nil {
		fb.err = err
		return
	}

	if fb.dir != "/" {
		fb.entries = append(fb.entries, fsEntry{name: "..", path: filepath.Dir(fb.dir), isDir: true, up: true})
	}

	var dirs, files []fsEntry
	for _, it := range items {
		name := it.Name()
		if !fb.hidden && strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(fb.dir, name)
		e := fsEntry{name: name, path: path}
		if it.IsDir() {
			e.isDir = true
			dirs = append(dirs, e)
			continue
		}
		if info, err := it.Info(); err == nil {
			e.size = info.Size()
		}
		files = append(files, e)
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })

	fb.entries = append(fb.entries, dirs...)
	fb.entries = append(fb.entries, files...)

	if keep >= len(fb.entries) {
		keep = len(fb.entries) - 1
	}
	if keep < 0 {
		keep = 0
	}
	fb.idx = keep
}

func (fb *fileBrowser) current() fsEntry {
	if fb.idx < 0 || fb.idx >= len(fb.entries) {
		return fsEntry{}
	}
	return fb.entries[fb.idx]
}

// selectValue is what gets handed back to the caller.
func (fb *fileBrowser) selectValue() (string, bool) {
	e := fb.current()
	if e.name == "" || e.up {
		return "", false
	}
	if fb.kind == android.ArgDir && !e.isDir {
		return "", false
	}
	if fb.kind == android.ArgFile && e.isDir {
		return "", false
	}
	return e.path, true
}

func (m *Model) updateFile(msg tea.Msg) (tea.Model, tea.Cmd) {
	fb := m.fb
	if fb == nil {
		m.pop()
		return m, nil
	}

	// The quick-jump field swallows keys while it is open.
	if fb.jumping {
		var cmd tea.Cmd
		m.ti, cmd = m.ti.Update(msg)
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "enter":
				target := strings.TrimSpace(m.ti.Value())
				fb.jumping = false
				m.ti.Blur()
				if target != "" {
					fb.jump(target)
				}
				return m, nil
			case "esc":
				fb.jumping = false
				m.ti.Blur()
				return m, nil
			}
		}
		return m, cmd
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "up", "k":
		if fb.idx > 0 {
			fb.idx--
		}
	case "down", "j":
		if fb.idx < len(fb.entries)-1 {
			fb.idx++
		}
	case "home", "g":
		fb.dir = sysinfo.Home()
		fb.reload(0)
	case "left", "h":
		parent := filepath.Dir(fb.dir)
		if parent != fb.dir {
			fb.dir = parent
			fb.reload(0)
		}
	case "right", "l", "d":
		if e := fb.current(); e.isDir {
			fb.dir = e.path
			fb.reload(0)
		}
	case "H":
		fb.hidden = !fb.hidden
		fb.reload(fb.idx)
	case "/":
		fb.jumping = true
		m.ti.Placeholder = i18n.Get("file.path")
		m.ti.SetValue(fb.dir)
		return m, m.ti.Focus()
	case "enter":
		e := fb.current()
		if e.up || e.name == "" {
			fb.idx = 0
			return m, nil
		}
		if e.isDir {
			// Entering a folder is the default; selecting it needs "s".
			fb.dir = e.path
			fb.reload(0)
			return m, nil
		}
		return m.acceptFile(e.path)
	case "s":
		if v, ok := fb.selectValue(); ok {
			return m.acceptFile(v)
		}
	case "esc":
		m.pop()
		return m, nil
	}
	return m, nil
}

// jump moves the browser to a path typed by the user, walking up until it finds
// a directory that exists.
func (fb *fileBrowser) jump(target string) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return
	}
	abs = filepath.Clean(abs)

	if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
		fb.dir = abs
		fb.reload(0)
		return
	}
	dir := filepath.Dir(abs)
	if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
		fb.dir = dir
		fb.reload(0)
		// Put the cursor on the entry the user typed.
		for i, e := range fb.entries {
			if e.name == filepath.Base(abs) {
				fb.idx = i
				break
			}
		}
	}
}

// acceptFile stores the chosen path and returns to the argument screen.
func (m *Model) acceptFile(path string) (tea.Model, tea.Cmd) {
	if m.fb == nil || len(m.pending.Args) == 0 {
		m.pop()
		return m, nil
	}
	arg := m.pending.Args[m.argIdx]
	m.argVals[arg.Key] = path
	m.notice = ""
	m.pop()

	// Refocus the text input in case the argument screen needs it again.
	if arg.Kind != android.ArgChoice {
		m.ti.SetValue(path)
	}
	return m, nil
}

func (m Model) viewFile() string {
	fb := m.fb
	if fb == nil {
		return ""
	}

	title := i18n.Get("file.pick.file")
	if fb.kind == android.ArgDir {
		title = i18n.Get("file.pick.dir")
	}

	body := []string{
		stHeader.Render(title),
		"",
		stLabel.Render(i18n.Get("file.path")+": ") + stValue.Render(fb.dir),
		"",
	}

	if fb.jumping {
		body = append(body, stLabel.Render(i18n.Get("cmd.need.file")), m.ti.View())
	}

	if fb.err != nil {
		body = append(body, "", stErr.Render(i18n.T("file.err.open", fb.err)))
	}

	if len(fb.entries) == 0 && fb.err == nil {
		body = append(body, "", stItemMuted.Render(i18n.Get("file.empty")))
	}

	// Keep the list inside the available height.
	maxRows := m.h - 16
	if maxRows < 5 {
		maxRows = 5
	}
	start, end := fb.window(maxRows)

	for i := start; i < end; i++ {
		e := fb.entries[i]
		sel := i == fb.idx
		row := fb.renderEntry(e, sel)
		if sel {
			body = append(body, row)
			continue
		}
		body = append(body, "  "+row)
	}

	if end < len(fb.entries) {
		body = append(body, stItemMuted.Render("  … "+i18n.T("file.size", itoa(len(fb.entries)-end))+" more"))
	}
	if start > 0 {
		body = append(body, stItemMuted.Render("  … "+itoa(start)))
	}

	box := boxStyle(m).Render(lipgloss.JoinVertical(lipgloss.Left, body...))

	foot := lipgloss.JoinVertical(lipgloss.Left,
		m.hint(keyCap("↑↓")+" — "+i18n.Get("common.select")+
			"  ·  "+keyCap("enter")+" — "+i18n.Get("file.enter.dir")+
			"  ·  "+keyCap("s")+" — "+i18n.Get("file.select.here")),
		m.hint(keyCap("←/h")+" — "+i18n.Get("file.up")+
			"  ·  "+keyCap("→/l")+" — "+i18n.Get("common.enter")+
			"  ·  "+keyCap("H")+" — "+i18n.Get("file.hidden")+
			"  ·  "+keyCap("g")+" — $HOME"+
			"  ·  "+keyCap("/")+" — path"+
			"  ·  "+keyCap("esc")+" — "+i18n.Get("common.cancel")),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header(title),
		"",
		box,
		"",
		foot,
	)
}

// nameWidth is the space available for the file name next to its size.
func (fb *fileBrowser) nameWidth() int {
	w := fb.viewW - 14
	return clamp(w, 12, 40)
}

func (fb *fileBrowser) window(maxRows int) (int, int) {
	total := len(fb.entries)
	if total <= maxRows {
		return 0, total
	}
	start := fb.idx - maxRows/2
	if start < 0 {
		start = 0
	}
	if start+maxRows > total {
		start = total - maxRows
	}
	return start, start + maxRows
}

func (fb *fileBrowser) renderEntry(e fsEntry, sel bool) string {
	mark := "  "
	if sel {
		mark = stCursor.Render("▸ ")
	}
	nameW := fb.nameWidth()

	switch {
	case e.up:
		return mark + stDir.Render(padRight(i18n.Get("file.up"), nameW))
	case e.isDir:
		return mark + stDir.Render(padRight(e.name, nameW)) + stDirMark.Render("/")
	default:
		size := humanSize(e.size)
		return mark + stFile.Render(padRight(e.name, nameW)) + stItemMuted.Render(size)
	}
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return itoa(int(n)) + " B"
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit && exp < 4; v /= unit {
		div *= unit
		exp++
	}
	suffix := [...]string{"K", "M", "G", "T", "P"}[exp]
	whole := n / div
	frac := (n % div) * 10 / div
	return itoa(int(whole)) + "." + itoa(int(frac)) + " " + suffix
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

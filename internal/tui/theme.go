package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"adm/internal/android"
	"adm/internal/i18n"
)

// Styles used across the interface.
var (
	stTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colAccent2)

	stSubtitle = lipgloss.NewStyle().
			Faint(true).
			Foreground(colMuted)

	stHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colAccent)

	stLabel = lipgloss.NewStyle().
		Foreground(colMuted)

	stValue = lipgloss.NewStyle().
		Foreground(colText)

	stOK = lipgloss.NewStyle().
		Foreground(colSuccess)

	stBad = lipgloss.NewStyle().
		Foreground(colDanger)

	stWarn = lipgloss.NewStyle().
		Foreground(colWarning)

	stItem = lipgloss.NewStyle().
		Foreground(colText)

	stItemMuted = lipgloss.NewStyle().
			Foreground(colMuted)

	stSel = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#11111b"}).
		Background(colAccent).
		Padding(0, 1)

	stSelEmpty = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	stRule = lipgloss.NewStyle().Foreground(colOverlay)

	stCursor = lipgloss.NewStyle().
			Bold(true).
			Foreground(colAccent)

	stHint = lipgloss.NewStyle().
		Faint(true).
		Foreground(colMuted)

	stErr = lipgloss.NewStyle().
		Bold(true).
		Foreground(colDanger)

	stBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colOverlay).
		Padding(1, 2)

	stBoxAccent = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colAccent).
			Padding(1, 2)

	stLogCmd = lipgloss.NewStyle().
			Bold(true).
			Foreground(colAccent)

	stLogErr = lipgloss.NewStyle().
			Foreground(colDanger)

	stCode = lipgloss.NewStyle().
		Foreground(colSuccess)

	stDanger = lipgloss.NewStyle().
			Bold(true).
			Foreground(colDanger)

	stGood = lipgloss.NewStyle().
		Bold(true).
		Foreground(colSuccess)

	stDir = lipgloss.NewStyle().
		Bold(true).
		Foreground(colAccent)

	stDirMark = lipgloss.NewStyle().
			Foreground(colAccent)

	stFile = lipgloss.NewStyle().
		Foreground(colText)
)

// lockedStyle renders the locked bootloader badge. A locked device is the safe
// case, so it gets the green colour.
func lockedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(colSuccess)
}

// unlockedStyle renders the unlocked bootloader badge, which is the risky case
// and therefore red.
func unlockedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(colDanger)
}

// badge renders the bootloader state word in its semantic colour: green for a
// locked device, red for an unlocked one.
func badge(state android.BootState, lockedWord, unlockedWord, unknownWord string) string {
	switch state {
	case android.BootLocked:
		return lockedStyle().Render(lockedWord)
	case android.BootUnlocked:
		return unlockedStyle().Render(unlockedWord)
	default:
		return stItemMuted.Render(unknownWord)
	}
}

// hint renders a key hint, wrapping it to the terminal width so nothing is cut
// off at the right edge. The text is taken literally: translated strings may
// contain percent signs, so it is never treated as a format string.
func (m Model) hint(s string) string {
	return stHint.Render(wrapText(s, m.w))
}

// selector renders one list row: a highlighted pill when selected, a plain row
// when not. The text is padded by hand rather than through Style.Width, because
// a fixed width would make lipgloss wrap anything longer than it.
func selector(selected bool, width int, text string) string {
	padded := padRight(text, width)
	if selected {
		return stSel.Render(padded)
	}
	return stSelEmpty.Render(padded)
}

// boxStyle returns the standard box. It keeps its natural width so short
// screens do not stretch across the terminal; long prose is wrapped by para
// instead, and a hard width here would pad every line with trailing spaces.
func boxStyle(m Model) lipgloss.Style {
	return stBox
}

// accentBox is the highlighted variant used for prompts.
func accentBox(m Model) lipgloss.Style {
	return stBoxAccent
}

// para wraps a block of prose to the width available inside a box.
func (m Model) para(s string) string {
	return stItemMuted.Render(wrapText(s, m.w-8))
}

// wrapText breaks text on word boundaries so it fits in width cells, keeping
// the blank lines that separate paragraphs.
//
// lipgloss's own Width() would pad every wrapped line out to the full width,
// and lipgloss.JoinVertical then stretches the whole frame to match, so the
// wrapping is done here instead.
func wrapText(s string, width int) string {
	if width < 20 {
		width = 20
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		if strings.TrimSpace(para) == "" {
			out = append(out, "")
			continue
		}
		out = append(out, wrapParagraph(para, width))
	}
	return strings.Join(out, "\n")
}

// wrapParagraph wraps one run of words, preserving any indentation it starts
// with so lists stay readable.
func wrapParagraph(s string, width int) string {
	indent := s[:len(s)-len(strings.TrimLeft(s, " \t"))]
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	cur := indent + words[0]
	for _, w := range words[1:] {
		if lipgloss.Width(cur)+1+lipgloss.Width(w) > width {
			lines = append(lines, cur)
			cur = indent + w
			continue
		}
		cur += " " + w
	}
	return strings.Join(append(lines, cur), "\n")
}

// header renders the persistent top bar with the application title.
func header(title string) string {
	line := stHeader.Render(title) + "  " + stTitle.Render(appName)
	return line
}

// rule draws a horizontal line of the given width using box-drawing glyphs.
func rule(width int) string {
	if width < 1 {
		return ""
	}
	if width > 200 {
		width = 200
	}
	out := make([]rune, width)
	for i := range out {
		out[i] = '─'
	}
	return stRule.Render(string(out))
}

// center places content horizontally and vertically in the given space.
func center(w, h int, content string) string {
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}

// keyCap renders a key hint as a small accent-coloured key name.
func keyCap(k string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(colAccent).Render(k)
}

// kv renders a "label: value" pair with the label dimmed.
func kv(label, value string) string {
	return stLabel.Render(label+": ") + stValue.Render(value)
}

// truncate shortens s to width, appending an ellipsis when it does not fit.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// padRight pads s with spaces up to width cells.
func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + spaces(width-w)
}

// clamp keeps v inside [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]rune, n)
	for i := range out {
		out[i] = ' '
	}
	return string(out)
}

// blockedBox shows why a command was refused.
func (m Model) blockedBox() string {
	if m.blocked == "" {
		return ""
	}
	return boxStyle(m).Render(lipgloss.JoinVertical(lipgloss.Left,
		stErr.Render("✗ "+i18n.Get("cmd.script.title")),
		"",
		stItem.Render(wrapText(m.blocked, m.w-12)),
		"",
		keyCap("esc")+" — "+i18n.Get("common.back"),
	))
}

// isTerminalFn is the seam the tests replace, because whether stdin happens to
// be a terminal depends on how the test binary was launched.
var isTerminalFn = isTerminal

// isTerminal reports whether standard input is an interactive terminal. A
// non-interactive run must never block waiting for a keypress.
func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

package tui

import (
	"context"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"adm/internal/android"
	"adm/internal/i18n"
)

const appName = "Android Device Management (TUI)"

// version and author are shown on the about screen and in --version. Both can
// be overridden at build time with -ldflags.
var (
	version = "1.0.1"
	author  = "grayfox951"
)

// screen identifies the active view.
type screen int

const (
	screenPreflight screen = iota
	screenDevices
	screenMenu
	screenCommands
	screenArgs
	screenFile
	screenOutput
	screenLang
	screenAbout
)

// confirmKind says which warning an inline confirmation is showing.
type confirmKind int

const (
	confirmNone confirmKind = iota
	confirmRisk
	confirmLocked
	confirmQuit
)

// tcpMode says which network prompt is on screen, if any.
type tcpMode int

const (
	tcpClosed tcpMode = iota
	tcpMenu
	tcpConnect
	tcpPairHost
	tcpPairCode
)

// tcpActions are the entries of the network panel, in display order.
var tcpActions = []struct {
	ID  string
	Key string
}{
	{"connect", "dev.tcp.action.connect"},
	{"pair", "dev.tcp.action.pair"},
	{"disconnect", "dev.tcp.action.disconnect"},
}

// Start values let main choose the entry screen.
const (
	// StartLang shows the language picker first.
	StartLang = screenLang
	// StartPreflight skips the picker and checks the environment at once.
	StartPreflight = screenPreflight
)

// SetupOutcome reports how the environment check ended.
type SetupOutcome int

const (
	// SetupOK means adb and fastboot are present.
	SetupOK SetupOutcome = iota
	// SetupDeclined means the user chose not to install platform-tools.
	SetupDeclined
	// SetupFailed means the install was attempted and did not work.
	SetupFailed
)

// ExitCode maps the outcome onto a process exit status for install.sh.
func (o SetupOutcome) ExitCode() int {
	switch o {
	case SetupDeclined:
		return 2
	case SetupFailed:
		return 3
	default:
		return 0
	}
}

// Model is the single bubbletea model behind every screen.
type Model struct {
	prog *tea.Program
	w, h int

	screen screen
	stack  []screen

	lang i18n.Lang
	// langIdx is the cursor of the language picker.
	langIdx int

	quitting bool
	err      error
	notice   string
	// blocked holds the refusal a command guard produced; any key dismisses it.
	blocked string

	// setupOnly runs the environment check and then quits instead of
	// carrying on to the device list. install.sh drives the program in this
	// mode so the platform-tools prompt looks exactly like the one in the
	// program itself.
	setupOnly bool
	outcome   SetupOutcome
	// awaitingSetupExit holds the finished setup screen on the terminal until
	// the user acknowledges it, so the result is actually readable.
	awaitingSetupExit bool

	// preflight
	pre preflight

	// devices
	devices  []android.Device
	device   android.Device
	devIdx   int
	scanning bool
	scanErr  error
	haveDev  bool
	showFull bool
	// tcpMode is what the network panel is currently asking for.
	tcpMode tcpMode
	tcpMenu int
	tcpHost string
	tcpBusy bool
	_       struct{}

	// menu
	cats   []string
	catIdx int

	// command list
	cat  string
	list []android.Command
	// fullToggle adds an extra row offering the unrestricted command list,
	// which is what makes a locked bootloader manageable.
	fullToggle bool
	listIdx    int

	// argument collection
	pending   android.Command
	argVals   map[string]string
	argIdx    int
	ti        textinput.Model
	pickingCh bool
	chIdx     int

	// inline confirmation
	confirm confirmKind

	// file browser
	fb *fileBrowser

	// running command output
	run runState

	sp spinner.Model
}

// Version returns the build version.
func Version() string { return version }

// Author returns the program's author.
func Author() string { return author }

// Err exposes the error that ended the session, if any.
func (m Model) Err() error { return m.err }

// Outcome reports how the setup session ended.
func (m Model) Outcome() SetupOutcome { return m.outcome }

// New builds the root model, detecting the interface language from the locale.
func New(lang string) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colAccent)

	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 0
	ti.Width = 60

	m := Model{
		lang:    pickLang(lang),
		screen:  screenLang,
		stack:   []screen{},
		sp:      sp,
		ti:      ti,
		argVals: map[string]string{},
	}
	m.applyLanguage()
	return m
}

// pickLang resolves an explicit language code, falling back to the locale.
func pickLang(code string) i18n.Lang {
	if code != "" {
		base := strings.ToLower(strings.SplitN(code, ".", 2)[0])
		base = strings.SplitN(base, "_", 2)[0]
		for _, l := range i18n.Order {
			if string(l) == base {
				return l
			}
		}
	}
	return detectLang()
}

// detectLang reads the locale from the environment.
func detectLang() i18n.Lang {
	raw := ""
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
		if v := strings.ToLower(os.Getenv(k)); v != "" {
			raw = v
			break
		}
	}
	switch {
	case strings.HasPrefix(raw, "ru"):
		return i18n.RU
	case strings.HasPrefix(raw, "uk"):
		return i18n.UK
	case strings.HasPrefix(raw, "be"):
		return i18n.BE
	case strings.HasPrefix(raw, "de"):
		return i18n.DE
	case strings.HasPrefix(raw, "en"):
		return i18n.EN
	}
	return i18n.EN
}

// applyLanguage pushes the chosen language into the i18n package.
func (m *Model) applyLanguage() {
	i18n.SetLanguage(m.lang)
	m.ti.CharLimit = 0
}

// push remembers the current screen so esc can return to it.
func (m *Model) push(s screen) {
	m.stack = append(m.stack, m.screen)
	m.screen = s
}

// pop returns to the previous screen.
func (m *Model) pop() screen {
	if len(m.stack) == 0 {
		return m.screen
	}
	m.screen = m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]
	return m.screen
}

// backTo unwinds the navigation stack until the given screen is on top.
func (m *Model) backTo(target screen) {
	m.screen = target
	m.stack = nil
}

// confirmsLocked reports whether the command is going to be run against a
// locked bootloader even though it normally needs an unlocked one.
func (m Model) confirmsLocked(c android.Command) bool {
	return m.device.Boot != android.BootUnlocked &&
		(c.Gate == android.GateUnlocked || c.Root)
}

// lockedLimited reports whether the currently selected device is in the
// restricted mode: locked bootloader, safe commands only.
func (m Model) lockedLimited() bool {
	return m.haveDev && m.device.Boot != android.BootUnlocked && !m.showFull
}

// contextWithTimeout is the default budget for a single adb or fastboot call
// made from the interface thread.
func contextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultTimeout)
}

// ---------------------------------------------------------------- messages

type devicesMsg struct {
	devices []android.Device
	err     error
}

type tickMsg struct{}

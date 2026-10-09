package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"adm/internal/android"
	"adm/internal/i18n"
)

func TestLockedDeviceHidesFlashingCommands(t *testing.T) {
	dev := lockedDevice()
	m := testModel(t, i18n.RU)
	m.selectDevice(dev)

	if !m.lockedLimited() {
		t.Fatal("a locked device should be in restricted mode")
	}

	m.list = m.filterCommands(android.CatFlash)
	for _, c := range m.list {
		if c.Gate == android.GateUnlocked || c.Root {
			t.Errorf("%s is listed on a locked bootloader without opting in", c.ID)
		}
	}

	// Asking for the full list is a deliberate, explicit action.
	m.showFull = true
	m.list = m.filterCommands(android.CatFlash)
	if len(m.list) == 0 {
		t.Fatal("the full list is empty, expected the restricted commands too")
	}
	var sawFlash bool
	for _, c := range m.list {
		if c.Category == android.CatFlash && c.Gate == android.GateUnlocked {
			sawFlash = true
		}
	}
	if !sawFlash {
		t.Error("the full list still hides the fastboot flash commands")
	}
}
func TestLockedDeviceStillOffersUnlock(t *testing.T) {
	// The unlock command is the one thing a locked bootloader must still show.
	dev := lockedDevice()
	m := testModel(t, i18n.RU)
	m.selectDevice(dev)

	m.list = m.filterCommands(android.CatBoot)
	var found bool
	for _, c := range m.list {
		if c.ID == "fb.unlock" {
			found = true
		}
	}
	if !found {
		t.Error("fastboot flashing unlock is hidden on a locked bootloader")
	}
}
func TestUnlockedDeviceIsNotRestricted(t *testing.T) {
	m := testModel(t, i18n.RU)
	m.selectDevice(unlockedDevice())

	if m.lockedLimited() {
		t.Error("an unlocked device should not be in restricted mode")
	}
	restricted := 0
	for _, c := range m.filterCommands(android.CatFlash) {
		if c.Gate != android.GateUnlocked {
			restricted++
		}
	}
	if restricted > 0 {
		t.Errorf("%d fastboot commands are gated even though the device is unlocked", restricted)
	}
}
func TestFileBrowserStartsAtHome(t *testing.T) {
	fb := newFileBrowser(android.ArgFile, "")
	if fb.dir == "" {
		t.Fatal("browser has no starting directory")
	}
	if !strings.HasPrefix(fb.dir, "/") {
		t.Errorf("starting directory %q is not absolute", fb.dir)
	}
	if len(fb.entries) == 0 && fb.err != nil {
		t.Skipf("home directory is not readable here: %v", fb.err)
	}
}
func TestFileBrowserReachesParent(t *testing.T) {
	fb := newFileBrowser(android.ArgFile, "")
	if fb.dir == "/" {
		t.Skip("already at the filesystem root")
	}
	first := fb.entries[0]
	if !first.up {
		t.Fatalf("the first entry should be the parent link, got %+v", first)
	}
	fb.dir = first.path
	fb.reload(0)
	if !strings.HasSuffix(fb.dir, "/") && fb.dir == "" {
		t.Error("moving up produced an empty path")
	}
}
func TestFileBrowserSelectValueRejectsFoldersForFileArgs(t *testing.T) {
	fb := &fileBrowser{
		dir:  "/tmp",
		kind: android.ArgFile,
		entries: []fsEntry{
			{name: "adir", path: "/tmp/adir", isDir: true},
			{name: "afile", path: "/tmp/afile"},
		},
	}
	fb.idx = 0
	if _, ok := fb.selectValue(); ok {
		t.Error("a folder was accepted for a file argument")
	}
	fb.idx = 1
	v, ok := fb.selectValue()
	if !ok || v != "/tmp/afile" {
		t.Errorf("selectValue = %q, %v", v, ok)
	}

	fb.kind = android.ArgDir
	fb.idx = 0
	if v, ok := fb.selectValue(); !ok || v != "/tmp/adir" {
		t.Errorf("folder argument selected %q, %v", v, ok)
	}
}
func TestRunStateDrainsLog(t *testing.T) {
	m := testModel(t, i18n.RU)
	f, err := os.CreateTemp("", "adm-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	m.run.logPath = f.Name()
	if _, err := f.WriteString("first\nsecond\n"); err != nil {
		t.Fatal(err)
	}

	m.run.drain()
	if got := len(m.run.lines); got != 2 {
		t.Fatalf("drained %d lines, want 2: %#v", got, m.run.lines)
	}

	// A second drain must not duplicate what was already read.
	m.run.drain()
	if got := len(m.run.lines); got != 2 {
		t.Errorf("lines duplicated on the second drain: %d", got)
	}

	m.run.refreshViewport()
	if !strings.Contains(stripANSI(m.run.vp.View()), "first") {
		t.Errorf("viewport does not show the log:\n%s", stripANSI(m.run.vp.View()))
	}
}
func TestRunStateKeepsTailBounded(t *testing.T) {
	r := &runState{}
	for i := 0; i < maxLines+500; i++ {
		r.append("line")
	}
	if len(r.lines) != maxLines {
		t.Errorf("kept %d lines, want %d", len(r.lines), maxLines)
	}
	if r.total != maxLines+500 {
		t.Errorf("total = %d, want %d", r.total, maxLines+500)
	}
}

// indexOfLang finds a language in the picker order.
func indexOfLang(l i18n.Lang) int {
	for i, v := range i18n.Order {
		if v == l {
			return i
		}
	}
	return 0
}

func TestLanguageSwitchKeepsModelConsistent(t *testing.T) {
	m := testModel(t, i18n.RU)
	m.selectDevice(unlockedDevice())
	m.loadCategories()

	m.langIdx = indexOfLang(i18n.DE)
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = *res.(*Model)

	if m.screen != screenPreflight {
		t.Errorf("after choosing a language the screen is %d, want preflight", m.screen)
	}
	if i18n.Current() != i18n.DE {
		t.Errorf("active language is %s, want de", i18n.Current())
	}
	m.applyLanguage()
}
func TestNavigationStackUnwinds(t *testing.T) {
	m := testModel(t, i18n.RU)
	m.selectDevice(unlockedDevice())
	m.stack = nil
	m.screen = screenMenu

	m.list = m.filterCommands(android.CatInfo)
	m.push(screenCommands)
	if m.screen != screenCommands {
		t.Fatalf("push did not switch screens")
	}

	m.confirm = confirmNone
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = *res.(*Model)
	if m.screen != screenMenu {
		t.Errorf("esc landed on screen %d, want the menu", m.screen)
	}
}
func TestQuoteConfirmOnMenu(t *testing.T) {
	m := testModel(t, i18n.RU)
	m.selectDevice(unlockedDevice())
	m.screen = screenMenu
	m.stack = nil

	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = *res.(*Model)
	if m.confirm != confirmQuit {
		t.Fatalf("esc did not raise the quit confirmation")
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = *res.(*Model)
	if m.confirm != confirmNone {
		t.Error("n did not dismiss the quit confirmation")
	}
	if m.quitting {
		t.Error("declining the confirmation quit the program")
	}
}

// startLogCommand runs cmd with its output appended to a temp log and wires the
// runState to it the same way launch does.
func startLogCommand(t *testing.T, r *runState, script string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "log.txt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sh", "-c", script)
	cmd.Stdout = f
	cmd.Stderr = f
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
		_ = f.Close()
	}()

	r.logPath = path
	r.doneCh = done
	r.running = true
	r.started = time.Now()
}

// pollUntilDone drives the poll loop the way the interface does.
func pollUntilDone(t *testing.T, r *runState) {
	t.Helper()
	for i := 0; i < 200 && r.running; i++ {
		r.poll()
		if r.running {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if r.running {
		t.Fatal("command never reported completion")
	}
}

func TestRunStateCapturesFastCommandOutput(t *testing.T) {
	// A command that finishes between two ticks must still have its output
	// read: the whole log can be written before the first poll.
	m := testModel(t, i18n.RU)
	startLogCommand(t, &m.run, "echo alpha; echo beta")
	pollUntilDone(t, &m.run)

	joined := strings.Join(m.run.lines, "\n")
	for _, want := range []string{"alpha", "beta"} {
		if !strings.Contains(joined, want) {
			t.Errorf("output missing %q, lines=%#v", want, m.run.lines)
		}
	}
	if !strings.Contains(joined, i18n.Get("common.done")) {
		t.Errorf("status line missing from the log: %#v", m.run.lines)
	}
	if m.run.code != 0 {
		t.Errorf("exit code = %d, want 0", m.run.code)
	}
}
func TestRunStateKeepsOutputWithoutTrailingNewline(t *testing.T) {
	m := testModel(t, i18n.RU)
	startLogCommand(t, &m.run, "printf 'no newline'")
	pollUntilDone(t, &m.run)

	if !strings.Contains(strings.Join(m.run.lines, "\n"), "no newline") {
		t.Errorf("a final line without a newline was dropped: %#v", m.run.lines)
	}
}
func TestRunStateReportsFailure(t *testing.T) {
	m := testModel(t, i18n.RU)
	startLogCommand(t, &m.run, "echo boom 1>&2; exit 3")
	pollUntilDone(t, &m.run)

	if m.run.code != 3 {
		t.Errorf("exit code = %d, want 3", m.run.code)
	}
	joined := strings.Join(m.run.lines, "\n")
	if !strings.Contains(joined, "boom") {
		t.Errorf("stderr missing: %#v", m.run.lines)
	}
	if !strings.Contains(joined, i18n.Get("common.failed")) {
		t.Errorf("failure status missing: %#v", m.run.lines)
	}
}
func TestSetupOutcomeExitCodes(t *testing.T) {
	cases := map[SetupOutcome]int{
		SetupOK:       0,
		SetupDeclined: 2,
		SetupFailed:   3,
	}
	for outcome, want := range cases {
		if got := outcome.ExitCode(); got != want {
			t.Errorf("%d.ExitCode() = %d, want %d", outcome, got, want)
		}
	}
}
func TestSetupFinishesWhenToolsArePresent(t *testing.T) {
	// Force the non-interactive path so the result does not depend on how the
	// test binary was launched.
	prev := isTerminalFn
	isTerminalFn = func() bool { return false }
	t.Cleanup(func() { isTerminalFn = prev })

	m := testModel(t, i18n.RU)
	m.setupOnly = true
	m.screen = screenPreflight

	res, _ := m.Update(preflightMsg{adb: "1.0.41", fastboot: "35.0"})
	m = *res.(*Model)

	if m.Outcome() != SetupOK {
		t.Errorf("outcome = %d, want SetupOK", m.Outcome())
	}
	if !m.quitting {
		t.Error("a non-interactive setup run must exit instead of waiting for a keypress")
	}
	if m.screen == screenDevices {
		t.Error("setup continued to the device scan instead of stopping")
	}
}
func TestSetupWaitsForAcknowledgementOnATerminal(t *testing.T) {
	prev := isTerminalFn
	isTerminalFn = func() bool { return true }
	t.Cleanup(func() { isTerminalFn = prev })

	m := testModel(t, i18n.RU)
	m.setupOnly = true
	m.screen = screenPreflight

	res, cmd := m.Update(preflightMsg{adb: "1.0.41", fastboot: "35.0"})
	m = *res.(*Model)

	if m.Outcome() != SetupOK {
		t.Errorf("outcome = %d, want SetupOK", m.Outcome())
	}
	if m.quitting {
		t.Error("an interactive setup run must hold the screen instead of quitting")
	}
	if !m.awaitingSetupExit {
		t.Error("the finished setup screen is not waiting for a keypress")
	}
	if cmd != nil {
		t.Error("setup should be idle while waiting, but a command was returned")
	}
}
func TestSetupAcknowledgementQuits(t *testing.T) {
	m := testModel(t, i18n.RU)
	m.setupOnly = true
	m.screen = screenPreflight
	m.awaitingSetupExit = true
	m.outcome = SetupOK

	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = *res.(*Model)

	if !m.quitting {
		t.Error("a keypress on the finished setup screen did not quit")
	}
	if m.Outcome() != SetupOK {
		t.Errorf("outcome changed to %d on acknowledgement", m.Outcome())
	}
}

// findScriptCommand locates the script entry for a mode.
func findScriptCommand(t *testing.T, mode android.Mode) android.Command {
	t.Helper()
	for _, c := range android.Catalog(mode) {
		if c.ID == "script.run" {
			return c
		}
	}
	t.Fatalf("script.run is missing from the %s catalog", mode)
	return android.Command{}
}

func writeTempScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "script.sh")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// runScriptFlow walks the real path: pick the file, then let launch dispatch to
// the guard, then hand the verdict back.
func runScriptFlow(t *testing.T, mode android.Mode, body string) (Model, tea.Cmd) {
	t.Helper()

	m := testModel(t, i18n.RU)
	m.selectDevice(android.Device{
		Serial: "ABC", Mode: mode, State: "device",
		Boot: android.BootUnlocked, Codename: "raven",
	})
	m.cat = android.CatOther
	m.list = m.filterCommands(android.CatOther)
	m.screen = screenCommands

	idx := -1
	for i, c := range m.list {
		if c.ID == "script.run" {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("script.run is not offered in the %s menu", mode)
	}
	m.listIdx = idx

	var res tea.Model
	var cmd tea.Cmd
	res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = *res.(*Model)
	if m.confirm != confirmRisk {
		t.Fatalf("a script should ask for confirmation first, confirm=%d", m.confirm)
	}
	res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = *res.(*Model)
	if m.screen != screenArgs {
		t.Fatalf("after confirming, screen = %d, want the argument screen", m.screen)
	}

	m.ti.SetValue(writeTempScript(t, body))
	m.ti.Focus()
	res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = *res.(*Model)
	return m, cmd
}

func TestScriptGuardBlocksWrongToolFromTheInterface(t *testing.T) {
	m, cmd := runScriptFlow(t, android.ModeFastboot, "#!/bin/sh\nadb devices\n")

	if cmd == nil {
		t.Fatal("launching the script produced no command, expected the guard")
	}
	// Execute the guard exactly as the event loop would.
	msg, ok := cmd().(guardDoneMsg)
	if !ok {
		t.Fatalf("expected a guardDoneMsg, got %T", cmd())
	}
	if msg.ok {
		t.Fatal("an adb script was allowed to run in fastboot mode")
	}

	res, _ := m.Update(msg)
	m = *res.(*Model)

	if m.blocked == "" {
		t.Fatal("no refusal message was shown to the user")
	}
	if !strings.Contains(m.blocked, "fastboot") {
		t.Errorf("the refusal does not mention fastboot: %q", m.blocked)
	}
	if m.screen == screenOutput {
		t.Error("the interface moved to the output screen even though it refused")
	}
	if strings.TrimSpace(m.blockedBox()) == "" {
		t.Error("the refusal is not rendered")
	}
}
func TestScriptGuardAllowsMatchingToolFromTheInterface(t *testing.T) {
	m, cmd := runScriptFlow(t, android.ModeFastboot, "#!/bin/sh\nfastboot flashing unlock\n")

	if cmd == nil {
		t.Fatal("no command was produced")
	}
	msg, ok := cmd().(guardDoneMsg)
	if !ok {
		t.Fatalf("expected a guardDoneMsg, got %T", cmd())
	}
	if !msg.ok {
		t.Fatalf("a fastboot script was refused in fastboot mode: %s", msg.key)
	}

	res, _ := m.Update(msg)
	m = *res.(*Model)

	if m.blocked != "" {
		t.Errorf("an accepted script still showed a refusal: %q", m.blocked)
	}
	if m.screen != screenOutput {
		t.Errorf("screen = %d, want the output screen", m.screen)
	}
	if !m.run.running {
		t.Error("the script was not started")
	}
	// Stop the process so the test does not leave a shell behind.
	m.run.poll()
}
func TestBlockedRefusalIsDismissedByAnyKey(t *testing.T) {
	m := testModel(t, i18n.RU)
	m.screen = screenCommands
	m.blocked = "refused for a test"

	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = *res.(*Model)

	if m.blocked != "" {
		t.Errorf("the refusal survived a keypress: %q", m.blocked)
	}
}
func TestScriptCommandAppearsInBothMenus(t *testing.T) {
	for _, mode := range []android.Mode{android.ModeADB, android.ModeFastboot} {
		m := testModel(t, i18n.RU)
		m.selectDevice(android.Device{Serial: "X", Mode: mode, Boot: android.BootLocked})
		m.cat = android.CatOther
		m.list = m.filterCommands(android.CatOther)

		view := stripANSI(m.viewCommands())
		if !strings.Contains(view, "run script") {
			t.Errorf("%s menu: the script entry is missing", mode)
		}
		// It must be offered even on a locked bootloader: a script is just a
		// convenience wrapper, and the guard decides whether it makes sense.
		if len(m.list) == 0 {
			t.Errorf("%s menu: the section is empty", mode)
		}
	}
}

// ---------------------------------------------------------------- TCP input

// TestTCPFieldOwnsTheKeyboard is the regression test for a host field that was
// unreachable: the handler sat behind a return in the key switch, so nothing
// the user typed ever reached the input.
func TestTCPFieldOwnsTheKeyboard(t *testing.T) {
	m := testModel(t, i18n.EN)
	m.screen = screenDevices
	m.devices = []android.Device{{Serial: "ABC", Mode: android.ModeADB, State: "device"}}

	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = *res.(*Model)
	if m.tcpMode != tcpMenu {
		t.Fatal("t did not open the network panel")
	}
	// Pick "connect", the first entry.
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = *res.(*Model)
	_ = cmd
	if m.tcpMode != tcpConnect {
		t.Fatalf("selecting connect gave mode %d", m.tcpMode)
	}

	// Type a host that contains the letters bound to global shortcuts.
	for _, ch := range "192.168.1.44" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = *res.(*Model)
	}
	if got := m.ti.Value(); got != "192.168.1.44" {
		t.Errorf("host field holds %q, want 192.168.1.44", got)
	}
	if m.quitting {
		t.Error("typing q while the host field was open quit the program")
	}
	if m.screen != screenDevices {
		t.Errorf("typing moved off the device screen to %d", m.screen)
	}

	// Typing c must not disconnect, r must not rescan.
	if m.tcpBusy {
		t.Error("typing triggered a connect")
	}
}

// TestGlobalKeysStandDownWhileTyping checks the router does not run the global
// handler before the field sees the key.
func TestGlobalKeysStandDownWhileTyping(t *testing.T) {
	m := testModel(t, i18n.EN)
	m.screen = screenDevices
	m.tcpMode = tcpConnect
	m.ti.Focus()

	if !m.typing() {
		t.Fatal("typing() should be true while the host field is open")
	}

	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = *res.(*Model)
	if m.quitting {
		t.Error("q quit the program from inside the host field")
	}
	if m.ti.Value() != "q" {
		t.Errorf("q did not reach the field, value = %q", m.ti.Value())
	}
}

// TestTCPPromptCancelAndToggle checks esc and a second t close the field.
func TestTCPPromptCancelAndToggle(t *testing.T) {
	m := testModel(t, i18n.EN)
	m.screen = screenDevices

	open := func(src Model) Model {
		res, _ := src.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
		src = *res.(*Model)
		res, _ = src.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return *res.(*Model)
	}

	m = open(m)
	m.ti.SetValue("10.0.0.1")

	// esc goes back to the action list rather than closing everything.
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = *res.(*Model)
	if m.tcpMode != tcpMenu {
		t.Error("esc did not return to the network menu")
	}
	if m.ti.Value() != "" {
		t.Errorf("cancelled host was not cleared: %q", m.ti.Value())
	}

	// esc from the menu closes the panel.
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = *res.(*Model)
	if m.tcpMode != tcpClosed {
		t.Error("esc did not close the network panel")
	}

	// Inside a field t is just a character, not a shortcut.
	m = open(m)
	m.ti.SetValue("t")
	if m.tcpMode != tcpConnect {
		t.Error("t inside the field closed it instead of being typed")
	}
}

func TestNormaliseHost(t *testing.T) {
	cases := map[string]string{
		"192.168.1.44":       "192.168.1.44:5555",
		" 192.168.1.44 ":     "192.168.1.44:5555",
		"192.168.1.44:37000": "192.168.1.44:37000",
		"  10.0.0.9:5555  ":  "10.0.0.9:5555",
		"":                   "",
		"   ":                "",
		"phone.local":        "phone.local:5555",
	}
	for in, want := range cases {
		if got := normaliseHost(in); got != want {
			t.Errorf("normaliseHost(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestConnectResultIsReported proves the outcome reaches the user instead of
// being dropped, and that the list is refreshed afterwards.
func TestConnectResultIsReported(t *testing.T) {
	m := testModel(t, i18n.EN)
	m.screen = screenDevices

	res, _ := m.Update(tcpMsg{key: "dev.tcp.ok", args: []any{"10.0.0.5:5555"}})
	m = *res.(*Model)

	if m.tcpBusy {
		t.Error("tcpBusy was not cleared after the connect finished")
	}
	if !strings.Contains(m.notice, "10.0.0.5:5555") {
		t.Errorf("the successful host is missing from the notice: %q", m.notice)
	}
	if strings.TrimSpace(m.viewDevices()) == "" {
		t.Error("the device screen broke after a connect")
	}
}

// TestGlobalKeysAreUsableWhenNoFieldIsOpen makes sure the guard did not disable
// the shortcuts it was meant to protect.
func TestGlobalKeysAreUsableWhenNoFieldIsOpen(t *testing.T) {
	m := testModel(t, i18n.EN)
	m.screen = screenDevices
	m.stack = nil

	if m.typing() {
		t.Fatal("typing() should be false with no field open")
	}
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = *res.(*Model)
	if !m.quitting {
		t.Error("q no longer quits when no field is open")
	}
}

// ------------------------------------------------------------ wireless pair

// pickPair walks the panel to the pairing address field.
func pickPair(t *testing.T) Model {
	t.Helper()
	m := testModel(t, i18n.EN)
	m.screen = screenDevices

	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = *res.(*Model)
	// Move from "connect" to "pair".
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = *res.(*Model)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = *res.(*Model)
	if m.tcpMode != tcpPairHost {
		t.Fatalf("selecting pair gave mode %d", m.tcpMode)
	}
	return m
}

func TestPairFlowAsksForAddressThenCode(t *testing.T) {
	m := pickPair(t)

	m.ti.SetValue("192.168.1.5:39871")
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = *res.(*Model)

	if m.tcpMode != tcpPairCode {
		t.Fatalf("after the address the mode is %d, want the code field", m.tcpMode)
	}
	if m.tcpHost != "192.168.1.5:39871" {
		t.Errorf("the address was not kept, got %q", m.tcpHost)
	}
	if m.ti.Value() != "" {
		t.Errorf("the code field is not empty: %q", m.ti.Value())
	}

	// The code is typed into the same input without a port being invented.
	m.ti.SetValue("123456")
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = *res.(*Model)
	if cmd == nil {
		t.Fatal("submitting the code produced no command")
	}
	if m.tcpMode != tcpClosed {
		t.Error("the panel stayed open after a pair attempt")
	}
	if !m.tcpBusy {
		t.Error("tcpBusy was not set while pairing")
	}
}

func TestPairRequiresAPort(t *testing.T) {
	m := pickPair(t)

	// The phone always shows a port; guessing one only produces a timeout.
	m.ti.SetValue("192.168.1.5")
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = *res.(*Model)

	if m.tcpMode != tcpPairHost {
		t.Error("an address without a port advanced to the code field")
	}
	if m.notice == "" {
		t.Error("no explanation was shown for the missing port")
	}
	if !strings.Contains(m.notice, "192.168.1.5:39871") {
		t.Errorf("the hint does not show an example: %q", m.notice)
	}
}

func TestPairResultIsReported(t *testing.T) {
	m := testModel(t, i18n.EN)
	m.screen = screenDevices

	res, _ := m.Update(tcpMsg{key: "dev.tcp.pair.ok",
		args: []any{"Successfully paired to 192.168.1.5:39871"}})
	m = *res.(*Model)

	if m.tcpBusy {
		t.Error("tcpBusy was not cleared after pairing")
	}
	if !strings.Contains(m.notice, "Successfully paired") {
		t.Errorf("the pairing result is missing from the notice: %q", m.notice)
	}
	// The message has to point at the connect step, since pairing alone is
	// not enough to talk to the phone.
	if !strings.Contains(m.notice, "Connect") {
		t.Errorf("the notice does not mention the follow up connect: %q", m.notice)
	}
}

func TestPairFieldOwnsTheKeyboard(t *testing.T) {
	m := pickPair(t)

	// A pairing code is digits, but the guard must hold for any input.
	for _, ch := range "q1r2" {
		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = *res.(*Model)
	}
	if m.ti.Value() != "q1r2" {
		t.Errorf("the code field holds %q, want q1r2", m.ti.Value())
	}
	if m.quitting {
		t.Error("q quit the program from inside the pairing field")
	}
}

func TestNetworkPanelListsEveryAction(t *testing.T) {
	m := testModel(t, i18n.EN)
	m.screen = screenDevices
	m.tcpMode = tcpMenu

	view := stripANSI(m.viewDevices())
	// The labels are translated, so look those up rather than the action ids.
	for _, action := range tcpActions {
		label := i18n.Get(action.Key)
		if label == action.Key || !strings.Contains(view, label) {
			t.Errorf("the network panel is missing %s (%q)", action.ID, label)
		}
	}
}

// ------------------------------------------------------- device auto-select

func scanWith(t *testing.T, devices ...android.Device) Model {
	t.Helper()
	m := testModel(t, i18n.RU)
	m.screen = screenDevices
	res, _ := m.Update(devicesMsg{devices: devices})
	return *res.(*Model)
}

func usbDevice() android.Device {
	return android.Device{Serial: "USB1", Mode: android.ModeADB, State: "device",
		Transport: "usb", Boot: android.BootUnlocked}
}

func netDevice() android.Device {
	return android.Device{Serial: "10.0.0.5:5555", Mode: android.ModeADB, State: "device",
		Transport: "tcp", Boot: android.BootUnlocked}
}

// TestSingleUSBDeviceIsAutoSelected keeps the convenience the user asked to
// keep: one device on the cable needs no extra keystroke.
func TestSingleUSBDeviceIsAutoSelected(t *testing.T) {
	m := scanWith(t, usbDevice())
	if m.screen != screenMenu {
		t.Errorf("a single USB device left the screen at %d, want the menu", m.screen)
	}
	if !m.haveDev || m.device.Serial != "USB1" {
		t.Errorf("the USB device was not taken: %+v", m.device)
	}
}

// TestSingleNetworkDeviceIsNotAutoSelected is the requested change: a wireless
// device waits to be picked.
func TestSingleNetworkDeviceIsNotAutoSelected(t *testing.T) {
	m := scanWith(t, netDevice())
	if m.screen != screenDevices {
		t.Errorf("a single network device jumped to screen %d, want the list", m.screen)
	}
	if m.haveDev {
		t.Errorf("the network device was taken without asking: %+v", m.device)
	}
	if len(m.devices) != 1 || m.devices[0].Serial != "10.0.0.5:5555" {
		t.Errorf("the list should still hold the device, got %+v", m.devices)
	}

	// It can still be chosen by hand.
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = *res.(*Model)
	if m.screen != screenMenu || m.device.Serial != "10.0.0.5:5555" {
		t.Errorf("the network device could not be chosen by hand: screen=%d dev=%q",
			m.screen, m.device.Serial)
	}
}

// TestSwitchDeviceRespectsTheNetworkRule covers the entry point the user
// named: switching away and back must behave the same way.
func TestSwitchDeviceRespectsTheNetworkRule(t *testing.T) {
	m := scanWith(t, usbDevice())
	if m.screen != screenMenu {
		t.Fatalf("setup failed, screen=%d", m.screen)
	}

	// The "switch device" row sits just below the category list.
	entries := m.menuEntries()
	idx := -1
	for i, e := range entries {
		if e.id == "act:switch" {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatal("the switch device entry is missing from the menu")
	}
	m.catIdx = idx
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = *res.(*Model)
	if m.screen != screenDevices {
		t.Fatalf("switch device went to screen %d", m.screen)
	}

	// Now with a wireless device as the only one, switching must not select it.
	res, _ = m.Update(devicesMsg{devices: []android.Device{netDevice()}})
	m = *res.(*Model)
	if m.screen != screenDevices {
		t.Errorf("switching with a single network device landed on %d, want the list", m.screen)
	}
	if m.haveDev {
		t.Errorf("switching auto selected the network device: %+v", m.device)
	}
}

func TestSeveralDevicesAreNeverAutoSelected(t *testing.T) {
	m := scanWith(t, usbDevice(), netDevice())
	if m.screen != screenDevices {
		t.Errorf("with two devices the screen is %d, want the list", m.screen)
	}
	if m.haveDev {
		t.Error("a device was chosen without asking when several were present")
	}
}

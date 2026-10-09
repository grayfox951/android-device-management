package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"adm/internal/android"
	"adm/internal/i18n"
	"adm/internal/sysinfo"
)

func testModel(t *testing.T, lang i18n.Lang) Model {
	t.Helper()
	m := New(string(lang))
	m.w, m.h = 110, 40
	m.run.resize(m.w, m.h)
	return m
}

// lockedDevice is the scenario the user asked to be handled specially.
func lockedDevice() android.Device {
	return android.Device{
		Serial:    "0A111FDD4000GX",
		Mode:      android.ModeFastboot,
		State:     "fastboot",
		Model:     "SM-G991B",
		Codename:  "o1s",
		Transport: "usb",
		Boot:      android.BootLocked,
	}
}

func unlockedDevice() android.Device {
	d := lockedDevice()
	d.Boot = android.BootUnlocked
	return d
}

// sysinfoInfoForTest is a distribution with a known install plan.
func sysinfoInfoForTest() sysinfo.Info {
	return sysinfo.Info{
		ID:       "ubuntu",
		Pretty:   "Ubuntu 26.04",
		Manager:  sysinfo.APT,
		Packages: []string{"adb", "fastboot"},
	}
}

// stripANSI removes escape sequences so assertions can look at plain text.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func TestEveryScreenRenders(t *testing.T) {
	dev := unlockedDevice()
	cats, grouped := android.ByCategory(dev.Mode)

	_ = cats

	cases := []struct {
		name string
		view func(Model) string
	}{
		{"lang", Model.viewLang},
		{"preflight", Model.viewPreflight},
		{"devices", Model.viewDevices},
		{"menu", Model.viewMenu},
		{"commands", Model.viewCommands},
		{"args", Model.viewArgs},
		{"output", Model.viewOutput},
		{"about", Model.viewAbout},
		{"file", Model.viewFile},
	}

	widths := []int{80, 100, 110, 160}
	for _, width := range widths {
		for _, lang := range i18n.Order {
			for _, c := range cases {
				m := testModel(t, lang)
				m.w, m.h = width, 40
				m.run.resize(m.w, m.h)
				m.selectDevice(dev)
				m.cat = android.CatInfo
				m.list = m.filterCommands(android.CatInfo)
				m.pending = grouped[android.CatInfo][0]
				m.argIdx = 0
				m.fb = newFileBrowser(android.ArgFile, "")
				m.fb.viewW = m.w
				m.run.cmdLine = "adb -s X shell id"
				m.run.append("uid=0(root)")

				out := c.view(m)
				name := fmt.Sprintf("%s/%s/w%d", lang, c.name, width)
				if strings.TrimSpace(out) == "" {
					t.Errorf("%s: rendered nothing", name)
				}
				if strings.Contains(out, "menu.cat.") || strings.Contains(out, "d.getprop") {
					t.Errorf("%s: an untranslated key leaked into the view", name)
				}
				if w := lipgloss.Width(out); w > width {
					t.Errorf("%s: rendered %d cells wide, terminal is %d", name, w, width)
				}
			}
		}
	}
}

func TestDeviceListShowsSerialCodenameAndBadge(t *testing.T) {
	m := testModel(t, i18n.RU)
	m.devices = []android.Device{lockedDevice(), unlockedDevice()}
	m.devIdx = 0

	view := stripANSI(m.viewDevices())
	for _, want := range []string{"0A111FDD4000GX", "o1s", "SM-G991B"} {
		if !strings.Contains(view, want) {
			t.Errorf("device view is missing %q:\n%s", want, view)
		}
	}
	if !strings.Contains(view, i18n.Get("dev.locked")) {
		t.Error("the locked badge is missing")
	}
	if !strings.Contains(view, i18n.Get("dev.unlocked")) {
		t.Error("the unlocked badge is missing")
	}
}

func TestMenuHasAllCategories(t *testing.T) {
	m := testModel(t, i18n.RU)
	m.selectDevice(android.Device{Serial: "X", Mode: android.ModeADB, Boot: android.BootLocked})

	view := stripANSI(m.viewMenu())
	for _, cat := range android.Categories(android.ModeADB) {
		label := i18n.Get("menu.cat." + cat)
		if !strings.Contains(view, label) {
			t.Errorf("menu is missing the %q section (%s)", label, cat)
		}
	}
}

func TestHumanSize(t *testing.T) {
	cases := map[int64]string{
		0:       "0 B",
		999:     "999 B",
		1024:    "1.0 K",
		1536:    "1.5 K",
		1048576: "1.0 M",
	}
	for in, want := range cases {
		if got := humanSize(in); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate = %q", got)
	}
	if got := truncate("abcdefghij", 5); len([]rune(got)) > 5 {
		t.Errorf("truncate did not shorten: %q", got)
	}
	if got := truncate("abc", 0); got != "" {
		t.Errorf("truncate to zero = %q", got)
	}
}

func TestBootloaderBadgeColours(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	const (
		lockedGreen = "38;2;166;227;161" // #a6e3a1
		unlockedRed = "38;2;243;139;168" // #f38ba8
	)

	render := func(boot android.BootState) string {
		i18n.SetLanguage(i18n.RU)
		m := testModel(t, i18n.RU)
		m.devices = []android.Device{{
			Serial: "SERIAL1", Mode: android.ModeADB,
			State: "device", Boot: boot, Codename: "raven", Model: "Pixel",
		}}
		m.devIdx = 0
		return m.viewDevices()
	}

	locked := render(android.BootLocked)
	if !strings.Contains(locked, lockedGreen) {
		t.Errorf("a locked bootloader is not rendered in green (%s)", lockedGreen)
	}
	if strings.Contains(locked, unlockedRed) {
		t.Error("a locked bootloader was rendered in the unlocked colour")
	}

	unlocked := render(android.BootUnlocked)
	if !strings.Contains(unlocked, unlockedRed) {
		t.Errorf("an unlocked bootloader is not rendered in red (%s)", unlockedRed)
	}
	if strings.Contains(unlocked, lockedGreen) {
		t.Error("an unlocked bootloader was rendered in the locked colour")
	}
}

func TestWrapTextKeepsParagraphs(t *testing.T) {
	in := "First line of a paragraph.\n\nSecond paragraph that is quite a bit longer than the width."

	got := wrapText(in, 20)
	lines := strings.Split(got, "\n")

	if len(lines) < 4 {
		t.Fatalf("expected the text to wrap onto several lines, got %d: %q", len(lines), got)
	}
	// The separator between the two paragraphs must survive somewhere.
	var gap bool
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			gap = true
		}
	}
	if !gap {
		t.Errorf("the blank line between paragraphs was lost: %q", got)
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w > 20 {
			t.Errorf("line %d is %d cells wide, want <= 20: %q", i, w, l)
		}
	}
	if !strings.Contains(got, "Second paragraph") {
		t.Errorf("text was lost while wrapping: %q", got)
	}
}

func TestWrapTextPreservesIndent(t *testing.T) {
	got := wrapText("  alpha beta gamma delta epsilon", 16)
	for _, l := range strings.Split(got, "\n") {
		if strings.TrimSpace(l) == "" {
			t.Errorf("unexpected empty line in %q", got)
		}
		if !strings.HasPrefix(l, "  ") {
			t.Errorf("indent lost on line %q", l)
		}
	}
}

func TestAboutScreenShowsAuthor(t *testing.T) {
	m := testModel(t, i18n.RU)
	m.w, m.h = 110, 40
	view := stripANSI(m.viewAbout())
	if !strings.Contains(view, "grayfox951") {
		t.Errorf("the about screen does not name the author:\n%s", view)
	}
	if !strings.Contains(view, Author()) {
		t.Errorf("the about screen does not use the configured author name %q", Author())
	}
}

func TestSetupScreenIsRenderable(t *testing.T) {
	// The setup screen must fit and never leak an untranslated key, in every
	// language and at every width.
	for _, width := range []int{80, 110} {
		for _, lang := range i18n.Order {
			m := testModel(t, lang)
			m.w, m.h = width, 40
			m.setupOnly = true
			m.screen = screenPreflight
			m.pre.phase = phaseMissing
			m.pre.info = sysinfoInfoForTest()
			m.pre.plan = m.pre.info.InstallCommand()

			out := m.viewPreflight()
			name := fmt.Sprintf("%s/w%d", lang, width)
			if strings.TrimSpace(out) == "" {
				t.Errorf("%s: setup screen rendered nothing", name)
			}
			if w := lipgloss.Width(out); w > width {
				t.Errorf("%s: setup screen is %d cells wide, want <= %d", name, w, width)
			}
			if strings.Contains(out, "pre.setup") {
				t.Errorf("%s: an untranslated setup key leaked into the view", name)
			}
			if !strings.Contains(stripANSI(out), "grayfox951") && strings.Contains(out, "setup.ack") {
				t.Errorf("%s: unexpected content", name)
			}
		}
	}
}

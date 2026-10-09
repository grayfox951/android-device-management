package android

import (
	"strings"
	"testing"

	"adm/internal/i18n"
)

func testDevice() Device {
	return Device{
		Serial:    "ABC123",
		Mode:      ModeADB,
		State:     "device",
		Model:     "Pixel 6 Pro",
		Codename:  "raven",
		Transport: "usb",
		Boot:      BootLocked,
	}
}

func TestEveryCommandBuildsArgv(t *testing.T) {
	for _, mode := range []Mode{ModeADB, ModeFastboot} {
		for _, c := range Catalog(mode) {
			vals := map[string]string{}
			for _, a := range c.Args {
				vals[a.Key] = a.Label
				if a.Default != "" {
					vals[a.Key] = a.Default
				}
				for _, ch := range a.Choices {
					vals[a.Key] = ch.Value
				}
			}

			argv := c.Build(testDevice(), vals)
			if len(argv) == 0 {
				t.Errorf("%s: Build returned an empty argv", c.ID)
				continue
			}
			// argv[0] must be the program name, otherwise exec would try to
			// run the flag instead of the tool. The few commands that need a
			// shell chain opt out with Shell and are checked separately.
			wantBin := "adb"
			if mode == ModeFastboot {
				wantBin = "fastboot"
			}
			gotBin := wantBin
			if c.Shell {
				gotBin = "sh"
			}
			if argv[0] != gotBin {
				t.Errorf("%s: argv starts with %q, want %q (argv=%v)", c.ID, argv[0], gotBin, argv)
				continue
			}
			if len(argv) < 2 {
				t.Errorf("%s: argv has no operation: %v", c.ID, argv)
			}
			if c.Shell && c.Guard == nil && argv[0] == "sh" &&
				!strings.Contains(strings.Join(argv, " "), wantBin) {
				t.Errorf("%s: shells out but never calls %s: %v", c.ID, wantBin, argv)
			}
		}
	}
}

func TestCommandsAreDescribed(t *testing.T) {
	for _, mode := range []Mode{ModeADB, ModeFastboot} {
		for _, c := range Catalog(mode) {
			if c.DescKey == "" {
				t.Errorf("%s: no description key", c.ID)
				continue
			}
			for _, l := range i18n.Order {
				i18n.SetLanguage(l)
				got := i18n.Get(c.DescKey)
				if got == c.DescKey {
					t.Errorf("%s: key %q not translated in %s", c.ID, c.DescKey, l)
				}
			}
			i18n.SetLanguage(i18n.RU)
		}
	}
}

func TestCommandIDsAreUnique(t *testing.T) {
	// Uniqueness is per mode: script.run deliberately exists in both
	// catalogs, once per tool.
	for _, mode := range []Mode{ModeADB, ModeFastboot} {
		seen := map[string]bool{}
		for _, c := range Catalog(mode) {
			if seen[c.ID] {
				t.Errorf("%s catalog: duplicate command id %q", mode, c.ID)
			}
			seen[c.ID] = true
		}
	}
}

func TestArgKeysAreUniquePerCommand(t *testing.T) {
	for _, mode := range []Mode{ModeADB, ModeFastboot} {
		for _, c := range Catalog(mode) {
			seen := map[string]bool{}
			for _, a := range c.Args {
				if seen[a.Key] {
					t.Errorf("%s: duplicate arg key %q", c.ID, a.Key)
				}
				seen[a.Key] = true
				if a.Kind == ArgChoice && len(a.Choices) == 0 {
					t.Errorf("%s: arg %q is a choice with no options", c.ID, a.Key)
				}
			}
		}
	}
}

func TestByCategoryCoversEverything(t *testing.T) {
	for _, mode := range []Mode{ModeADB, ModeFastboot} {
		order, grouped := ByCategory(mode)
		total := 0
		for _, cat := range order {
			total += len(grouped[cat])
		}
		if total != len(Catalog(mode)) {
			t.Errorf("%s: ByCategory exposes %d commands, catalog has %d",
				mode, total, len(Catalog(mode)))
		}
	}
}

func TestServerRestartDoesBothHalves(t *testing.T) {
	// The label promises kill-server && start-server; issuing only the first
	// would leave the user without a working adb afterwards.
	var c Command
	for _, cmd := range Catalog(ModeADB) {
		if cmd.ID == "server.restart" {
			c = cmd
		}
	}
	if c.ID == "" {
		t.Fatal("server.restart is missing from the catalog")
	}

	argv := strings.Join(c.Build(Device{Serial: "X"}, map[string]string{}), " ")
	if !strings.Contains(argv, "kill-server") || !strings.Contains(argv, "start-server") {
		t.Errorf("server.restart does not do both halves: %q", argv)
	}
}

func TestDeviceArgvHelpers(t *testing.T) {
	d := testDevice()
	if got := strings.Join(d.ADB("shell", "id"), " "); got != "adb -s ABC123 shell id" {
		t.Errorf("ADB helper produced %q", got)
	}
	if got := strings.Join(d.Fastboot("getvar", "all"), " "); got != "fastboot -s ABC123 getvar all" {
		t.Errorf("Fastboot helper produced %q", got)
	}
}

func TestBootStateString(t *testing.T) {
	cases := map[BootState]string{
		BootLocked:   "locked",
		BootUnlocked: "unlocked",
		BootUnknown:  "unknown",
	}
	for state, want := range cases {
		if got := state.String(); got != want {
			t.Errorf("BootState(%d).String() = %q, want %q", state, got, want)
		}
	}
}

func TestParseGetvar(t *testing.T) {
	// fastboot writes getvar results to stderr in a "name: value" shape.
	out := "finished. total time: 0.001s\nunlocked: yes\nis-userspace: no\n"
	if got := parseGetvar(out, "unlocked"); got != "yes" {
		t.Errorf("parseGetvar(unlocked) = %q, want yes", got)
	}
	if got := parseGetvar(out, "missing"); got != "" {
		t.Errorf("parseGetvar(missing) = %q, want empty", got)
	}
}

func TestScanParsesDeviceLists(t *testing.T) {
	// The parsers are exercised through their output shapes so a change in
	// adb formatting shows up here rather than on screen.
	adbOut := `List of devices attached
ABC123       device product:raven model:Pixel_6_Pro device:raven transport_id:1
DEF456       unauthorized usb:1-1 transport_id:2
emulator-5554 device product:sdk_gphone device:emu transport_id:3

`
	if got := parseADBDevices(adbOut); len(got) != 3 {
		t.Fatalf("parsed %d adb devices, want 3", len(got))
	}
	if got := parseADBDevices(adbOut); got[0].Codename != "raven" || got[0].Model != "Pixel_6_Pro" {
		t.Errorf("first device = %+v", got[0])
	}
	if got := parseADBDevices(adbOut); got[2].Transport != "tcp" {
		t.Errorf("emulator transport = %q, want tcp", got[2].Transport)
	}
	if got := parseADBDevices(adbOut); got[1].State != "unauthorized" {
		t.Errorf("second device state = %q", got[1].State)
	}

	fbOut := "ABC123\tfastboot\nDEF456 fastboot\n\n"
	fb := parseFastbootDevices(fbOut)
	if len(fb) != 2 {
		t.Fatalf("parsed %d fastboot devices, want 2", len(fb))
	}
	if fb[0].Serial != "ABC123" || fb[1].Serial != "DEF456" {
		t.Errorf("fastboot serials = %q, %q", fb[0].Serial, fb[1].Serial)
	}
}

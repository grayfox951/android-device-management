package android

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"adm/internal/i18n"
)

func writeScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "script.sh")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func runScriptGuard(t *testing.T, mode Mode, body string) (bool, string) {
	t.Helper()

	path := writeScript(t, body)
	cmd := scriptCommand(mode)

	ctx, cancel := context.WithTimeout(context.Background(), scriptCheckTimeout)
	defer cancel()

	ok, key, args := cmd.Guard(ctx, Device{Serial: "X", Mode: mode}, map[string]string{"script": path})
	if ok {
		return true, ""
	}

	i18n.SetLanguage(i18n.EN)
	if got := i18n.T(key, args...); got == key {
		t.Errorf("refusal key %q has no translation", key)
	}
	return false, key
}

func TestScriptGuardAcceptsItsOwnTool(t *testing.T) {
	cases := []struct {
		mode Mode
		body string
	}{
		{ModeADB, "#!/bin/sh\nadb devices\n"},
		{ModeADB, "adb devices\n"},
		{ModeADB, "#!/bin/sh\n/usr/bin/adb shell id\n"},
		{ModeADB, "#!/bin/sh\nif [ -z \"$X\" ]; then\n  adb reboot\nfi\n"},
		{ModeADB, "#!/bin/sh\necho hi | adb shell cat\n"},
		{ModeFastboot, "#!/bin/sh\nfastboot devices\n"},
		{ModeFastboot, "#!/bin/sh\nfastboot flash boot boot.img\n"},
	}
	for _, c := range cases {
		if ok, key := runScriptGuard(t, c.mode, c.body); !ok {
			t.Errorf("mode %s: refused a valid script (%s):\n%s", c.mode, key, c.body)
		}
	}
}

func TestScriptGuardRejectsTheOtherTool(t *testing.T) {
	// The whole point of the feature: an adb script is useless while the phone
	// sits in fastboot, and vice versa.
	if ok, key := runScriptGuard(t, ModeFastboot, "#!/bin/sh\nadb devices\n"); ok {
		t.Error("an adb script was accepted in fastboot mode")
	} else if key != "cmd.script.tool.fastboot" {
		t.Errorf("refusal key = %q, want cmd.script.tool.fastboot", key)
	}

	if ok, key := runScriptGuard(t, ModeADB, "#!/bin/sh\nfastboot flashing unlock\n"); ok {
		t.Error("a fastboot script was accepted in adb mode")
	} else if key != "cmd.script.tool.adb" {
		t.Errorf("refusal key = %q, want cmd.script.tool.adb", key)
	}
}

func TestScriptGuardRejectsScriptWithoutAnyTool(t *testing.T) {
	bodies := []string{
		"",
		"#!/bin/sh\n# just a note\necho hello\n",
		"#!/bin/bash\nls -la\nmake\n",
		"#!/bin/sh\n# adb\n",
		"#!/bin/sh\n# fastboot devices\n",
	}
	for _, body := range bodies {
		if ok, key := runScriptGuard(t, ModeADB, body); ok {
			t.Errorf("accepted a script with no commands:\n%q", body)
		} else if key != "cmd.script.tool.adb" && key != "cmd.script.tool.fastboot" {
			t.Errorf("unexpected refusal key %q for %q", key, body)
		}
	}
}

func TestScriptGuardRejectsUnreadableFile(t *testing.T) {
	for _, mode := range []Mode{ModeADB, ModeFastboot} {
		cmd := scriptCommand(mode)
		ctx := context.Background()

		// Missing file.
		ok, key, _ := cmd.Guard(ctx, Device{Mode: mode}, map[string]string{"script": "/nope/missing.sh"})
		if ok || key != "cmd.script.catfail" {
			t.Errorf("missing file: ok=%v key=%q", ok, key)
		}

		// Nothing chosen at all.
		ok, key, _ = cmd.Guard(ctx, Device{Mode: mode}, map[string]string{"script": "  "})
		if ok || key != "cmd.script.nofile" {
			t.Errorf("empty path: ok=%v key=%q", ok, key)
		}
	}
}

func TestScriptCommandBuildsShellArgv(t *testing.T) {
	for _, mode := range []Mode{ModeADB, ModeFastboot} {
		c := scriptCommand(mode)
		argv := c.Build(Device{Mode: mode}, map[string]string{"script": "/tmp/x.sh"})
		if len(argv) != 2 || argv[0] != "sh" || argv[1] != "/tmp/x.sh" {
			t.Errorf("mode %s: argv = %v", mode, argv)
		}
		if !c.Risk {
			t.Errorf("mode %s: a script must be confirmed as a risky operation", mode)
		}
	}
}

func TestScriptCommandIsInBothCatalogs(t *testing.T) {
	for _, mode := range []Mode{ModeADB, ModeFastboot} {
		var found bool
		for _, c := range Catalog(mode) {
			if c.ID == "script.run" {
				found = true
				if c.Mode != mode {
					t.Errorf("script.run in the %s catalog reports mode %s", mode, c.Mode)
				}
			}
		}
		if !found {
			t.Errorf("script.run is missing from the %s catalog", mode)
		}
	}
}

func TestMentionsToolIsNotFooledBySubstrings(t *testing.T) {
	// A path or variable that merely contains the word must not count.
	for _, body := range []string{
		"cat /home/user/adb-backup.txt\n",
		"TOTAL=5\n",
		"echo noadbcommands\n",
	} {
		if mentionsTool(body, "adb") {
			t.Errorf("mentionsTool(adb) matched %q", body)
		}
	}
	if !mentionsTool("echo x; adb devices\n", "adb") {
		t.Error("mentionsTool missed a real command followed by ;")
	}
	if !mentionsTool("/opt/fastboot devices\n", "fastboot") {
		t.Error("mentionsTool missed an absolute path invocation")
	}
	// A shebang names the interpreter, never the tool being scripted.
	if mentionsTool("#!/usr/bin/env adb\n", "adb") {
		t.Error("mentionsTool must not count a shebang as a tool call")
	}
	if mentionsTool("# adb devices\n#fastboot flash\n", "adb") {
		t.Error("mentionsTool must not count a comment as a tool call")
	}
}

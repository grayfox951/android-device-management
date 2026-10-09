package android

import (
	"context"
	"strings"
	"time"

	"adm/internal/runner"
)

// scriptCheckTimeout bounds the `cat` used to inspect a script before running
// it, so a huge or unreadable file cannot freeze the interface.
const scriptCheckTimeout = 15 * time.Second

// scriptCommand builds the "run a prepared script" entry for one of the two
// tools. The same command exists in both catalogs but each variant insists the
// script actually mentions its own tool: running an adb script against a phone
// sitting in fastboot would only produce a pile of "device not found".
func scriptCommand(mode Mode) Command {
	tool := "adb"
	toolKey := "cmd.script.tool.adb"
	if mode == ModeFastboot {
		tool = "fastboot"
		toolKey = "cmd.script.tool.fastboot"
	}

	return Command{
		ID:       "script.run",
		Category: CatOther,
		Mode:     mode,
		Label:    "run script with " + tool + " commands",
		DescKey:  "d.script.run",
		Args:     []Arg{file("script")},
		// A script is arbitrary code and may well flash the device, so it goes
		// through the risk confirmation like a destructive command would.
		Risk:  true,
		Shell: true,
		Build: func(d Device, a map[string]string) []string {
			// sh, not bash: a script has to run even where bash is absent, and
			// plain adb/fastboot one-liners need nothing more.
			return []string{"sh", a["script"]}
		},
		Guard: guardScriptMentions(tool, toolKey),
	}
}

// guardScriptMentions returns a Guard that reads the picked file with cat and
// refuses to continue when the file never mentions the tool for the current
// mode.
func guardScriptMentions(tool, toolKey string) Guard {
	return func(ctx context.Context, d Device, a map[string]string) (bool, string, []any) {
		path := strings.TrimSpace(a["script"])
		if path == "" {
			return false, "cmd.script.nofile", nil
		}

		checkCtx, cancel := context.WithTimeout(ctx, scriptCheckTimeout)
		defer cancel()

		res := runner.RunQuiet(checkCtx, "cat", path)
		if res.Err != nil {
			return false, "cmd.script.catfail", []any{res.Err}
		}
		if res.Code != 0 {
			return false, "cmd.script.catfail", []any{strings.TrimSpace(res.Out)}
		}

		if !mentionsTool(res.Out, tool) {
			return false, toolKey, nil
		}
		return true, "", nil
	}
}

// mentionsTool reports whether the script body actually invokes the given
// tool. Comments are skipped, so a script that only talks *about* adb is not
// mistaken for one that uses it, and the tool name has to stand as a whole
// word so paths and variable names do not count either.
func mentionsTool(body, tool string) bool {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// A shebang names the interpreter and a comment says nothing is run.
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if lineInvokesTool(trimmed, tool) {
			return true
		}
	}
	return false
}

// lineInvokesTool looks for the tool as a command word on a single line.
func lineInvokesTool(line, tool string) bool {
	fields := strings.Fields(line)
	for i, f := range fields {
		// A leading FOO=bar is an assignment, not a command.
		if i == 0 && strings.Contains(f, "=") && !strings.HasSuffix(f, ";") {
			continue
		}
		word := strings.TrimRight(f, ";&|")
		word = strings.Trim(word, `"'`)
		if word == tool {
			return true
		}
		// adb and fastboot are also invoked through a full path.
		if strings.HasSuffix(word, "/"+tool) {
			return true
		}
	}
	return false
}

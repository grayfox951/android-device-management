// Command adm is a terminal interface for Android devices on Linux. It wraps
// adb and fastboot: it checks that the platform tools are installed, offers to
// install them for the detected distribution, lets the user pick a device and
// then exposes the command catalog through a menu.
//
// # Copyright (C) 2026 grayfox951
//
// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for
// more details.
//
// You should have received a copy of the GNU General Public License along with
// this program. If not, see <https://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: GPL-3.0-only
//
// Author: grayfox951
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"adm/internal/i18n"
	"adm/internal/sysinfo"
	"adm/internal/tui"
)

// Flags live at package scope so the helpers below can read them.
var (
	langFlag   = flag.String("lang", "", "interface language: ru, en, uk, be, de (default: from the locale)")
	listLangs  = flag.Bool("languages", false, "print the available interface languages and exit")
	checkOnly  = flag.Bool("check", false, "only report whether adb and fastboot are present")
	showVer    = flag.Bool("version", false, "print the version and exit")
	installNow = flag.Bool("install", false, "install platform-tools for the detected distribution and exit")
	setupMode  = flag.Bool("setup", false, "check adb and fastboot in the interface, install them if asked, then exit")
)

func main() {
	flag.Parse()

	if *showVer {
		fmt.Printf("Android Device Management (TUI) %s — %s\n", tui.Version(), tui.Author())
		return
	}

	if *listLangs {
		for _, l := range i18n.Order {
			fmt.Printf("%-3s %s\n", l, i18n.Names[l])
		}
		return
	}

	lang := resolveLang(*langFlag)
	i18n.SetLanguage(lang)

	if *checkOnly {
		os.Exit(checkTools())
	}
	if *installNow {
		os.Exit(installTools())
	}

	// install.sh drives the program through this mode so the platform-tools
	// prompt is the same screen the user sees on a normal run.
	if *setupMode {
		os.Exit(runSetup())
	}

	// An empty code lets the interface ask or fall back to the locale.
	langCode := ""
	if *langFlag != "" {
		langCode = string(resolveLang(*langFlag))
	}
	m, err := tui.Run(tui.StartLang, langCode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if m.Err() != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", m.Err())
		os.Exit(1)
	}
}

// runSetup performs the environment check through the interface and maps the
// result onto an exit status: 0 ready, 2 declined, 3 install failed.
func runSetup() int {
	langCode := ""
	if *langFlag != "" {
		langCode = string(resolveLang(*langFlag))
	}

	m, err := tui.RunSetup(langCode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	if m.Err() != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", m.Err())
		return 1
	}
	return m.Outcome().ExitCode()
}

// resolveLang turns the --lang flag into a supported language, falling back to
// the environment locale and finally to English.
func resolveLang(flagVal string) i18n.Lang {
	if flagVal != "" {
		for _, l := range i18n.Order {
			if strings.EqualFold(flagVal, string(l)) {
				return l
			}
		}
		// Accept the locale form, e.g. --lang ru_RU.UTF-8.
		base := strings.ToLower(strings.SplitN(flagVal, ".", 2)[0])
		base = strings.SplitN(base, "_", 2)[0]
		for _, l := range i18n.Order {
			if string(l) == base {
				return l
			}
		}
		fmt.Fprintf(os.Stderr, "unknown language %q, using English\n", flagVal)
		return i18n.EN
	}
	// Run() falls back to the locale when the picker is skipped.
	return i18n.EN
}

// checkTools reports the availability of the platform tools and exits.
func checkTools() int {
	var missing []string
	for _, t := range []string{"adb", "fastboot"} {
		if p := sysinfo.Which(t); p != "" {
			fmt.Printf("%-9s %s\n", t, p)
		} else {
			missing = append(missing, t)
			fmt.Printf("%-9s %s\n", t, i18n.Get("common.missing"))
		}
	}
	if len(missing) > 0 {
		fmt.Fprintln(os.Stderr, i18n.T("pre.missing", strings.Join(missing, ", ")))
		return 1
	}
	return 0
}

// installTools runs the distribution install command outside the interface and
// exits with its status.
func installTools() int {
	info := sysinfo.Detect()
	plan := info.InstallCommand()

	fmt.Println(i18n.T("pre.distro", info.Display()))
	if len(plan.Alternates) > 0 {
		fmt.Println(i18n.T("pre.alt", strings.Join(plan.Alternates, " · ")))
	}
	if plan.Command == "" {
		fmt.Fprintln(os.Stderr, i18n.Get("pre.nodetect"))
		return 1
	}
	fmt.Println("$ " + plan.Command)
	fmt.Println(i18n.Get("pre.sudo"))

	return runInstall(plan.Command)
}

// runInstall executes the install command attached to the real terminal so a
// sudo password prompt works.
func runInstall(line string) int {
	cmd := exec.Command("sh", "-c", line)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

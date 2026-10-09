package android

import "context"

// Categories group commands in the menu.
const (
	CatInfo     = "info"
	CatFiles    = "files"
	CatApps     = "apps"
	CatBoot     = "boot"
	CatFlash    = "flash"
	CatRecovery = "recovery"
	CatSystem   = "system"
	CatNetwork  = "network"
	CatDebug    = "debug"
	CatOther    = "other"
)

// Gate says whether a command can run on a locked bootloader.
type Gate int

const (
	// GateAny means the command is expected to work with a locked bootloader.
	GateAny Gate = iota
	// GateUnlocked means the command needs an unlocked bootloader.
	GateUnlocked
)

// ArgKind is the widget used to collect one argument.
type ArgKind int

const (
	ArgText ArgKind = iota
	ArgPackage
	ArgFile
	ArgDir
	ArgRemotePath
	ArgPartition
	ArgChoice
)

// ChoiceOpt is one entry of a fixed option list.
type ChoiceOpt struct {
	Value string
	Label string
}

// Arg is a single value the user has to supply before a command runs.
type Arg struct {
	Key     string
	Kind    ArgKind
	Label   string
	Default string
	Choices []ChoiceOpt
}

// Guard inspects the collected arguments before a command runs and can refuse
// it. Returning ok=false blocks the command; key is an i18n key naming the
// refusal and args are its format arguments.
type Guard func(ctx context.Context, d Device, a map[string]string) (ok bool, key string, args []any)

// Command is one entry of the catalog.
type Command struct {
	ID       string
	Category string
	// Label is the adb/fastboot syntax, shown verbatim because it is what the
	// user would type on a shell.
	Label string
	// DescKey points at the translated one-line description.
	DescKey string
	Mode    Mode
	Gate    Gate
	// Root marks commands that need an unlocked, rooted device.
	Root bool
	// Risk marks commands that destroy data, so they get an extra warning.
	Risk bool
	// Redirect names the arg whose value receives stdout instead of the
	// output screen, used for screencap and logcat dumps.
	Redirect string
	// Shell marks the few commands that need sh -c because a single tool
	// invocation cannot express what they do, such as restarting the adb
	// daemon. Their argv therefore starts with "sh", not with adb or fastboot.
	Shell bool
	Args  []Arg
	// Guard, when set, runs before Build and can block the command.
	Guard Guard
	// Build returns the complete argv, program name included.
	Build func(d Device, a map[string]string) []string
}

// AvailableOn reports whether the command can be offered for a device in the
// given bootloader state when only safe commands are being shown.
func (c Command) AvailableOn(boot BootState) bool {
	if c.Gate == GateUnlocked || c.Root {
		return boot == BootUnlocked
	}
	return true
}

// NeedsArgs reports whether the command collects any input before running.
func (c Command) NeedsArgs() bool { return len(c.Args) > 0 }

func txt(k, label, def string) Arg {
	return Arg{Key: k, Kind: ArgText, Label: label, Default: def}
}

func pkg() Arg {
	return Arg{Key: "package", Kind: ArgPackage, Label: "com.example.app"}
}

func file(k string) Arg {
	return Arg{Key: k, Kind: ArgFile, Label: "file"}
}

func dir(k string) Arg {
	return Arg{Key: k, Kind: ArgDir, Label: "folder"}
}

func remote(k string) Arg {
	return Arg{Key: k, Kind: ArgRemotePath, Label: "/sdcard/"}
}

func part(k string) Arg {
	return Arg{Key: k, Kind: ArgPartition, Label: "boot"}
}

func choice(k, label string, def string, opts ...ChoiceOpt) Arg {
	return Arg{Key: k, Kind: ArgChoice, Label: label, Default: def, Choices: opts}
}

func shellCmds(cmds ...string) []ChoiceOpt {
	out := make([]ChoiceOpt, 0, len(cmds))
	for _, c := range cmds {
		out = append(out, ChoiceOpt{Value: c, Label: c})
	}
	return out
}

func a(args []string, extra ...string) []string {
	return append(append([]string(nil), args...), extra...)
}

func sh(args ...string) []string {
	return a([]string{"shell"}, args...)
}

// AdbCommands is the catalog of everything reachable through adb.
func AdbCommands() []Command {
	var c []Command

	// ---------------------------------------------------------------- info
	c = append(c,
		Command{
			ID: "getprop", Category: CatInfo, Mode: ModeADB, Label: "getprop [key]",
			DescKey: "d.getprop",
			Args:    []Arg{choice("key", "property", "", shellCmds("ro.product.model", "ro.build.version.release", "ro.build.display.id", "ro.serialno", "ro.boot.slot_suffix", "ro.product.cpu.abi", "ro.build.type")...)},
			Build: func(d Device, v map[string]string) []string {
				if v["key"] == "" {
					return d.ADB("shell", "getprop")
				}
				return d.ADB("shell", "getprop", v["key"])
			},
		},
		Command{
			ID: "info.summary", Category: CatInfo, Mode: ModeADB,
			Label: "device summary", DescKey: "d.info.summary",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "sh", "-c",
					"echo model: $(getprop ro.product.model); echo device: $(getprop ro.product.device); echo android: $(getprop ro.build.version.release); echo build: $(getprop ro.build.display.id); echo abi: $(getprop ro.product.cpu.abi); echo slot: $(getprop ro.boot.slot_suffix); echo bootloader: $(getprop ro.boot.verifiedbootstate); echo sdk: $(getprop ro.build.version.sdk)")
			},
		},
		Command{
			ID: "info.battery", Category: CatInfo, Mode: ModeADB,
			Label: "dumpsys battery", DescKey: "d.info.battery",
			Build: func(d Device, v map[string]string) []string { return d.ADB("shell", "dumpsys", "battery") },
		},
		Command{
			ID: "info.meminfo", Category: CatInfo, Mode: ModeADB,
			Label: "dumpsys meminfo", DescKey: "d.info.meminfo",
			Build: func(d Device, v map[string]string) []string { return d.ADB("shell", "dumpsys", "meminfo") },
		},
		Command{
			ID: "info.df", Category: CatInfo, Mode: ModeADB,
			Label: "shell df -h", DescKey: "d.info.df",
			Build: func(d Device, v map[string]string) []string { return d.ADB("shell", "df", "-h") },
		},
		Command{
			ID: "info.cpu", Category: CatInfo, Mode: ModeADB,
			Label: "shell cat /proc/cpuinfo", DescKey: "d.info.cpu",
			Build: func(d Device, v map[string]string) []string { return d.ADB("shell", "cat", "/proc/cpuinfo") },
		},
		Command{
			ID: "info.kernel", Category: CatInfo, Mode: ModeADB,
			Label: "shell uname -a", DescKey: "d.info.kernel",
			Build: func(d Device, v map[string]string) []string { return d.ADB("shell", "uname", "-a") },
		},
		Command{
			ID: "info.uptime", Category: CatInfo, Mode: ModeADB,
			Label: "shell uptime", DescKey: "d.info.uptime",
			Build: func(d Device, v map[string]string) []string { return d.ADB("shell", "uptime") },
		},
		Command{
			ID: "info.display", Category: CatInfo, Mode: ModeADB,
			Label: "wm size / wm density", DescKey: "d.info.display",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "sh", "-c", "wm size; wm density")
			},
		},
		Command{
			ID: "info.ps", Category: CatInfo, Mode: ModeADB,
			Label: "shell ps -A", DescKey: "d.info.ps",
			Build: func(d Device, v map[string]string) []string { return d.ADB("shell", "ps", "-A") },
		},
		Command{
			ID: "info.features", Category: CatInfo, Mode: ModeADB,
			Label: "shell pm list features", DescKey: "d.info.features",
			Build: func(d Device, v map[string]string) []string { return d.ADB("shell", "pm", "list", "features") },
		},
	)

	// --------------------------------------------------------------- files
	c = append(c,
		Command{
			ID: "push", Category: CatFiles, Mode: ModeADB, Label: "push <file> <device path>",
			DescKey: "d.push",
			Args:    []Arg{file("local"), remote("remote")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("push", v["local"], v["remote"])
			},
		},
		Command{
			ID: "pull", Category: CatFiles, Mode: ModeADB, Label: "pull <device path> [folder]",
			DescKey: "d.pull",
			Args:    []Arg{remote("remote"), dir("local")},
			Build: func(d Device, v map[string]string) []string {
				if v["local"] == "" {
					return d.ADB("pull", v["remote"])
				}
				return d.ADB("pull", v["remote"], v["local"])
			},
		},
		Command{
			ID: "ls", Category: CatFiles, Mode: ModeADB, Label: "shell ls -la <path>",
			DescKey: "d.ls",
			Args:    []Arg{remote("path")},
			Build: func(d Device, v map[string]string) []string {
				p := v["path"]
				if p == "" {
					p = "/sdcard"
				}
				return d.ADB("shell", "ls", "-la", p)
			},
		},
		Command{
			ID: "du", Category: CatFiles, Mode: ModeADB, Label: "shell du -h <path>",
			DescKey: "d.du",
			Args:    []Arg{remote("path")},
			Build: func(d Device, v map[string]string) []string {
				p := v["path"]
				if p == "" {
					p = "/sdcard"
				}
				return d.ADB("shell", "du", "-h", p)
			},
		},
		Command{
			ID: "mkdir", Category: CatFiles, Mode: ModeADB, Label: "shell mkdir -p <path>",
			DescKey: "d.mkdir",
			Args:    []Arg{remote("path")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "mkdir", "-p", v["path"])
			},
		},
		Command{
			ID: "rm", Category: CatFiles, Mode: ModeADB, Label: "shell rm -rf <path>", Risk: true,
			DescKey: "d.rm",
			Args:    []Arg{remote("path")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "rm", "-rf", v["path"])
			},
		},
		Command{
			ID: "cat", Category: CatFiles, Mode: ModeADB, Label: "shell cat <file>",
			DescKey: "d.cat",
			Args:    []Arg{remote("path")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "cat", v["path"])
			},
		},
		Command{
			ID: "stat", Category: CatFiles, Mode: ModeADB, Label: "shell ls -ld <path>",
			DescKey: "d.stat",
			Args:    []Arg{remote("path")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "ls", "-ld", v["path"])
			},
		},
		Command{
			ID: "storage.info", Category: CatFiles, Mode: ModeADB,
			Label: "df / mounts / sdcard", DescKey: "d.storage.info",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "sh", "-c", "df -h; echo; mount | grep -E 'sdcard|fuse|emulated'; echo; ls -la /sdcard")
			},
		},
	)

	// ---------------------------------------------------------------- apps
	c = append(c,
		Command{
			ID: "pm.list", Category: CatApps, Mode: ModeADB, Label: "pm list packages",
			DescKey: "d.pm.list",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "pm", "list", "packages")
			},
		},
		Command{
			ID: "pm.list3", Category: CatApps, Mode: ModeADB, Label: "pm list packages -3",
			DescKey: "d.pm.list3",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "pm", "list", "packages", "-3")
			},
		},
		Command{
			ID: "pm.path", Category: CatApps, Mode: ModeADB, Label: "pm path <package>",
			DescKey: "d.pm.path", Args: []Arg{pkg()},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "pm", "path", v["package"])
			},
		},
		Command{
			ID: "install", Category: CatApps, Mode: ModeADB, Label: "install <apk>",
			DescKey: "d.install",
			Args:    []Arg{file("apk")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("install", v["apk"])
			},
		},
		Command{
			ID: "install.r", Category: CatApps, Mode: ModeADB, Label: "install -r -d -g <apk>",
			DescKey: "d.install.r",
			Args:    []Arg{file("apk")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("install", "-r", "-d", "-g", v["apk"])
			},
		},
		Command{
			ID: "install.multiple", Category: CatApps, Mode: ModeADB, Label: "install-multiple -r <apks…>",
			DescKey: "d.install.multiple",
			Args:    []Arg{file("apks")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("install-multiple", "-r", v["apks"])
			},
		},
		Command{
			ID: "uninstall", Category: CatApps, Mode: ModeADB, Label: "uninstall <package>", Risk: true,
			DescKey: "d.uninstall", Args: []Arg{pkg()},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("uninstall", v["package"])
			},
		},
		Command{
			ID: "pm.clear", Category: CatApps, Mode: ModeADB, Label: "pm clear <package>", Risk: true,
			DescKey: "d.pm.clear", Args: []Arg{pkg()},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "pm", "clear", v["package"])
			},
		},
		Command{
			ID: "pm.dump", Category: CatApps, Mode: ModeADB, Label: "dumpsys package <package>",
			DescKey: "d.pm.dump", Args: []Arg{pkg()},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "dumpsys", "package", v["package"])
			},
		},
		Command{
			ID: "pm.enable", Category: CatApps, Mode: ModeADB, Label: "pm enable / disable <package>",
			DescKey: "d.pm.enable",
			Args: []Arg{pkg(), choice("action", "action", "disable",
				ChoiceOpt{Value: "enable", Label: "enable"},
				ChoiceOpt{Value: "disable", Label: "disable"},
				ChoiceOpt{Value: "clear", Label: "clear"})},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "pm", v["action"], v["package"])
			},
		},
	)

	// ---------------------------------------------------------------- boot
	c = append(c,
		Command{
			ID: "reboot", Category: CatBoot, Mode: ModeADB, Label: "reboot",
			DescKey: "d.reboot",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("reboot")
			},
		},
		Command{
			ID: "reboot.bootloader", Category: CatBoot, Mode: ModeADB, Label: "reboot bootloader",
			DescKey: "d.reboot.bootloader",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("reboot", "bootloader")
			},
		},
		Command{
			ID: "reboot.recovery", Category: CatBoot, Mode: ModeADB, Label: "reboot recovery",
			DescKey: "d.reboot.recovery",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("reboot", "recovery")
			},
		},
		Command{
			ID: "reboot.sideload", Category: CatBoot, Mode: ModeADB, Label: "reboot sideload",
			DescKey: "d.reboot.sideload",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("reboot", "sideload")
			},
		},
		Command{
			ID: "reboot.edl", Category: CatBoot, Mode: ModeADB, Label: "reboot edl (download mode)",
			DescKey: "d.reboot.edl", Gate: GateUnlocked, Risk: true,
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("reboot", "edl")
			},
		},
		Command{
			ID: "remount", Category: CatBoot, Mode: ModeADB, Label: "remount",
			DescKey: "d.remount", Root: true,
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("remount")
			},
		},
		Command{
			ID: "stayawake", Category: CatBoot, Mode: ModeADB, Label: "svc power stayon true",
			DescKey: "d.stayawake",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "svc", "power", "stayon", "true")
			},
		},
		Command{
			ID: "root.restart", Category: CatBoot, Mode: ModeADB, Label: "adbd restart (root)", Root: true,
			DescKey: "d.root.restart",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("root")
			},
		},
	)

	// ------------------------------------------------------------ recovery
	c = append(c,
		Command{
			ID: "sideload", Category: CatRecovery, Mode: ModeADB, Label: "sideload <zip>",
			DescKey: "d.sideload",
			Args:    []Arg{file("zip")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("sideload", v["zip"])
			},
		},
		Command{
			ID: "sideload.session", Category: CatRecovery, Mode: ModeADB, Label: "sideload_session <zip>",
			DescKey: "d.sideload.session",
			Args:    []Arg{file("zip")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "sideload_session", v["zip"])
			},
		},
		Command{
			ID: "recovery.roots", Category: CatRecovery, Mode: ModeADB,
			Label: "su -c id / ls /", Root: true, DescKey: "d.recovery.roots",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "su", "-c", "id; echo; ls -la /")
			},
		},
		Command{
			ID: "recovery.wipe", Category: CatRecovery, Mode: ModeADB, Gate: GateUnlocked, Risk: true,
			Label: "recovery --wipe_data", DescKey: "d.recovery.wipe",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "su", "-c", "recovery --wipe_data")
			},
		},
		Command{
			ID: "recovery.reboot", Category: CatRecovery, Mode: ModeADB,
			Label: "su -c 'recovery --reboot'", Root: true, DescKey: "d.recovery.reboot",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "su", "-c", "recovery --reboot")
			},
		},
	)

	// -------------------------------------------------------------- system
	c = append(c,
		Command{
			ID: "shell", Category: CatSystem, Mode: ModeADB, Label: "shell <command>",
			DescKey: "d.shell",
			Args:    []Arg{txt("cmd", "command", "")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "sh", "-c", v["cmd"])
			},
		},
		Command{
			ID: "su.check", Category: CatSystem, Mode: ModeADB, Label: "su -c id", Root: true,
			DescKey: "d.su.check",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "su", "-c", "id")
			},
		},
		Command{
			ID: "settings.list", Category: CatSystem, Mode: ModeADB, Label: "settings list global/system/secure",
			DescKey: "d.settings.list",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "sh", "-c", "settings list global; echo; settings list system; echo; settings list secure")
			},
		},
		Command{
			ID: "settings.put", Category: CatSystem, Mode: ModeADB, Label: "settings put <namespace> <key> <value>",
			DescKey: "d.settings.put",
			Args: []Arg{
				choice("ns", "namespace", "global",
					ChoiceOpt{Value: "global", Label: "global"},
					ChoiceOpt{Value: "system", Label: "system"},
					ChoiceOpt{Value: "secure", Label: "secure"}),
				txt("key", "key", ""), txt("value", "value", ""),
			},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "settings", "put", v["ns"], v["key"], v["value"])
			},
		},
		Command{
			ID: "settings.get", Category: CatSystem, Mode: ModeADB, Label: "settings get <namespace> <key>",
			DescKey: "d.settings.get",
			Args: []Arg{
				choice("ns", "namespace", "global",
					ChoiceOpt{Value: "global", Label: "global"},
					ChoiceOpt{Value: "system", Label: "system"},
					ChoiceOpt{Value: "secure", Label: "secure"}),
				txt("key", "key", ""),
			},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "settings", "get", v["ns"], v["key"])
			},
		},
		Command{
			ID: "setprop", Category: CatSystem, Mode: ModeADB, Label: "setprop <key> <value>",
			DescKey: "d.setprop", Root: true,
			Args: []Arg{txt("key", "key", ""), txt("value", "value", "")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "su", "-c", "setprop "+v["key"]+" "+v["value"])
			},
		},
		Command{
			ID: "input.keyevent", Category: CatSystem, Mode: ModeADB, Label: "input keyevent <code>",
			DescKey: "d.input.keyevent",
			Args: []Arg{choice("code", "key", "KEYCODE_HOME",
				ChoiceOpt{Value: "KEYCODE_HOME", Label: "HOME"},
				ChoiceOpt{Value: "KEYCODE_BACK", Label: "BACK"},
				ChoiceOpt{Value: "KEYCODE_POWER", Label: "POWER"},
				ChoiceOpt{Value: "KEYCODE_VOLUME_UP", Label: "VOLUME UP"},
				ChoiceOpt{Value: "KEYCODE_VOLUME_DOWN", Label: "VOLUME DOWN"},
				ChoiceOpt{Value: "KEYCODE_CAMERA", Label: "CAMERA"},
				ChoiceOpt{Value: "KEYCODE_WAKEUP", Label: "WAKEUP"},
				ChoiceOpt{Value: "KEYCODE_SLEEP", Label: "SLEEP"},
				ChoiceOpt{Value: "KEYCODE_APP_SWITCH", Label: "RECENTS"},
				ChoiceOpt{Value: "KEYCODE_MENU", Label: "MENU"})},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "input", "keyevent", v["code"])
			},
		},
		Command{
			ID: "input.text", Category: CatSystem, Mode: ModeADB, Label: "input text <text>",
			DescKey: "d.input.text",
			Args:    []Arg{txt("text", "text", "hello")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "input", "text", v["text"])
			},
		},
		Command{
			ID: "input.tap", Category: CatSystem, Mode: ModeADB, Label: "input tap <x> <y>",
			DescKey: "d.input.tap",
			Args:    []Arg{txt("x", "x", "500"), txt("y", "y", "1000")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "input", "tap", v["x"], v["y"])
			},
		},
		Command{
			ID: "input.swipe", Category: CatSystem, Mode: ModeADB, Label: "input swipe <x1> <y1> <x2> <y2> [ms]",
			DescKey: "d.input.swipe",
			Args:    []Arg{txt("x1", "x1", "500"), txt("y1", "y1", "1500"), txt("x2", "x2", "500"), txt("y2", "y2", "300"), txt("ms", "duration ms", "300")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "input", "swipe", v["x1"], v["y1"], v["x2"], v["y2"], v["ms"])
			},
		},
		Command{
			ID: "wm.density", Category: CatSystem, Mode: ModeADB, Label: "wm density <dpi>",
			DescKey: "d.wm.density",
			Args:    []Arg{txt("dpi", "dpi", "420")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "wm", "density", v["dpi"])
			},
		},
		Command{
			ID: "wm.size", Category: CatSystem, Mode: ModeADB, Label: "wm size <WxH>",
			DescKey: "d.wm.size",
			Args:    []Arg{txt("size", "WxH", "1080x2400")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "wm", "size", v["size"])
			},
		},
		Command{
			ID: "pm.permissions", Category: CatSystem, Mode: ModeADB, Label: "dumpsys package <pkg> (permissions)",
			DescKey: "d.pm.permissions", Args: []Arg{pkg()},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "dumpsys", "package", v["package"])
			},
		},
		Command{
			ID: "am.start", Category: CatSystem, Mode: ModeADB, Label: "am start <component>",
			DescKey: "d.am.start",
			Args:    []Arg{txt("component", "component", "")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "am", "start", "-n", v["component"])
			},
		},
		Command{
			ID: "am.intent", Category: CatSystem, Mode: ModeADB, Label: "am start <action>/<data>",
			DescKey: "d.am.intent",
			Args:    []Arg{txt("uri", "action or uri", "")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "am", "start", "-a", v["uri"])
			},
		},
		Command{
			ID: "service.list", Category: CatSystem, Mode: ModeADB, Label: "shell service list",
			DescKey: "d.service.list",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "service", "list")
			},
		},
	)

	// ------------------------------------------------------------- network
	c = append(c,
		Command{
			ID: "net.tcpip", Category: CatNetwork, Mode: ModeADB, Label: "tcpip <port>",
			DescKey: "d.net.tcpip",
			Args:    []Arg{txt("port", "port", "5555")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("tcpip", v["port"])
			},
		},
		Command{
			ID: "net.forward", Category: CatNetwork, Mode: ModeADB, Label: "forward tcp:<local> tcp:<remote>",
			DescKey: "d.net.forward",
			Args:    []Arg{txt("local", "local", "tcp:8080"), txt("remote", "remote", "tcp:8080")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("forward", v["local"], v["remote"])
			},
		},
		Command{
			ID: "net.forward.list", Category: CatNetwork, Mode: ModeADB, Label: "forward --list",
			DescKey: "d.net.forward.list",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("forward", "--list")
			},
		},
		Command{
			ID: "net.forward.kill", Category: CatNetwork, Mode: ModeADB, Label: "forward --remove <local>",
			DescKey: "d.net.forward.kill",
			Args:    []Arg{txt("local", "local", "tcp:8080")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("forward", "--remove", v["local"])
			},
		},
		Command{
			ID: "net.reverse", Category: CatNetwork, Mode: ModeADB, Label: "reverse tcp:<remote> tcp:<local>",
			DescKey: "d.net.reverse",
			Args:    []Arg{txt("remote", "device port", "tcp:3000"), txt("local", "host port", "tcp:3000")},
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("reverse", v["remote"], v["local"])
			},
		},
		Command{
			ID: "net.reverse.list", Category: CatNetwork, Mode: ModeADB, Label: "reverse --list",
			DescKey: "d.net.reverse.list",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("reverse", "--list")
			},
		},
		Command{
			ID: "net.wifi.info", Category: CatNetwork, Mode: ModeADB, Label: "dumpsys wifi (head)",
			DescKey: "d.net.wifi.info",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "dumpsys", "wifi")
			},
		},
		Command{
			ID: "net.ip", Category: CatNetwork, Mode: ModeADB, Label: "shell ip addr / ip route",
			DescKey: "d.net.ip",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "sh", "-c", "ip -brief addr 2>/dev/null || ip addr; echo; ip route")
			},
		},
	)

	// --------------------------------------------------------------- debug
	c = append(c,
		Command{
			ID: "logcat", Category: CatDebug, Mode: ModeADB, Label: "logcat",
			DescKey: "d.logcat",
			Args:    []Arg{choice("buffer", "buffer", "", shellCmds("main", "system", "radio", "events", "crash", "all")...)},
			Build: func(d Device, v map[string]string) []string {
				if v["buffer"] == "" {
					return d.ADB("logcat")
				}
				return d.ADB("logcat", "-b", v["buffer"])
			},
		},
		Command{
			ID: "logcat.clear", Category: CatDebug, Mode: ModeADB, Label: "logcat -c",
			DescKey: "d.logcat.clear",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("logcat", "-c")
			},
		},
		Command{
			ID: "logcat.file", Category: CatDebug, Mode: ModeADB,
			Label: "logcat -d > <file>", DescKey: "d.logcat.file",
			Args:     []Arg{file("out")},
			Redirect: "out",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("logcat", "-d")
			},
		},
		Command{
			ID: "dumpsys", Category: CatDebug, Mode: ModeADB, Label: "dumpsys <service>",
			DescKey: "d.dumpsys",
			Args: []Arg{choice("service", "service", "",
				shellCmds("activity", "window", "package", "meminfo", "battery", "power", "wifi", "connectivity", "audio", "input", "alarm", "notification", "deviceidle", "usagestats", "display", "SurfaceFlinger")...)},
			Build: func(d Device, v map[string]string) []string {
				if v["service"] == "" {
					return d.ADB("shell", "dumpsys")
				}
				return d.ADB("shell", "dumpsys", v["service"])
			},
		},
		Command{
			ID: "bugreport", Category: CatDebug, Mode: ModeADB, Label: "bugreport",
			DescKey: "d.bugreport",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("bugreport")
			},
		},
		Command{
			ID: "debug.ports", Category: CatDebug, Mode: ModeADB, Label: "shell /proc/net/tcp + getprop",
			DescKey: "d.debug.ports",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("shell", "sh", "-c", "cat /proc/net/tcp 2>/dev/null | head -20; echo; getprop")
			},
		},
		Command{
			ID: "debug.track", Category: CatDebug, Mode: ModeADB, Label: "track-devices",
			DescKey: "d.debug.track",
			Build: func(d Device, v map[string]string) []string {
				return d.ADB("track-devices")
			},
		},
	)

	return c
}

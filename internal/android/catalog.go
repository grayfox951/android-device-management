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

	return c
}

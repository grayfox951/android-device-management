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

	return c
}

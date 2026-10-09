// Package sysinfo detects the host distribution and knows how to install
// Android platform-tools (adb + fastboot) on it.
package sysinfo

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
)

// Manager identifies a package manager.
type Manager string

// Unknown is the zero value so an Info that has not been classified yet
// reports itself as unrecognised.
const (
	Unknown Manager = ""
	Pacman  Manager = "pacman"
	APT     Manager = "apt"
	Dnf     Manager = "dnf"
)

// Info describes the detected distribution and the install plan for it.
type Info struct {
	Manager Manager
	// ID is the primary /etc/os-release ID, e.g. "arch", "ubuntu".
	ID string
	// Pretty is a human friendly name taken from PRETTY_NAME.
	Pretty string
	// Like holds ID_LIKE values, e.g. "debian", "arch".
	Like []string
	// Packages is the preferred package list.
	Packages []string
	// Alternates are fallback package names if the preferred ones are absent.
	Alternates []string
	// UdevPackages carry the udev rules needed for non-root USB access.
	UdevPackages []string
}

// Supported reports whether the distribution is one of the officially
// supported targets (arch, artix, mint, ubuntu, debian and derivatives).
func (i Info) Supported() bool {
	return i.Manager == Pacman || i.Manager == APT
}

// Known reports whether we recognised the distribution at all.
func (i Info) Known() bool {
	return i.Manager != Unknown && i.ID != ""
}

// Display returns the best available human readable distribution name.
func (i Info) Display() string {
	if i.Pretty != "" {
		return i.Pretty
	}
	if i.ID != "" {
		return i.ID
	}
	return "unknown"
}

// Plan is a shell command that installs platform-tools.
type Plan struct {
	// Command is the full shell command line, ready to be printed.
	Command string
	// Packages lists what will be installed.
	Packages []string
	// Alternates mentions fallback package names.
	Alternates []string
}

// InstallCommand builds the sudo command that installs platform-tools.
func (i Info) InstallCommand() Plan {
	pkgs := append([]string(nil), i.Packages...)
	switch i.Manager {
	case Pacman:
		pkgs = append(pkgs, i.UdevPackages...)
		return Plan{
			Command:    "sudo pacman -S --needed --noconfirm " + strings.Join(dedupe(pkgs), " "),
			Packages:   pkgs,
			Alternates: i.Alternates,
		}
	case APT:
		return Plan{
			Command:    "sudo apt-get install -y " + strings.Join(dedupe(pkgs), " "),
			Packages:   pkgs,
			Alternates: i.Alternates,
		}
	case Dnf:
		return Plan{
			Command:    "sudo dnf install -y " + strings.Join(dedupe(pkgs), " "),
			Packages:   pkgs,
			Alternates: i.Alternates,
		}
	default:
		return Plan{Alternates: i.Alternates}
	}
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// Detect reads /etc/os-release and, when the identifiers are inconclusive,
// falls back to probing for the package manager binaries.
func Detect() Info {
	id, like, pretty := readOSRelease()

	info := classify(id, like)
	info.Pretty = pretty

	if info.Manager == Unknown {
		info.Manager = probeManager()
	}
	info.applyPackages()
	return info
}

// classify resolves the distribution purely from its identifiers, without
// touching the host, which keeps the mapping table testable.
func classify(id string, like []string) Info {
	var info Info

	info.ID = id
	info.Like = like

	// Prefer the concrete ID, fall back to the first ID_LIKE entry.
	switch {
	case id == "arch" || id == "artix":
		info.Manager = Pacman
	case id == "mint":
		info.Manager = APT
	case id == "ubuntu" || id == "debian" || id == "pop" || id == "elementary" ||
		id == "kali" || id == "raspbian" || id == "deepin" || id == "zorin":
		info.Manager = APT
	}

	if info.Manager == Unknown {
		for _, l := range like {
			switch l {
			case "arch", "artix", "manjaro", "endeavouros":
				info.Manager = Pacman
			case "debian", "ubuntu":
				info.Manager = APT
			}
			if info.Manager != Unknown {
				break
			}
		}
	}

	info.applyPackages()
	return info
}

// probeManager looks for a package manager binary on the host.
func probeManager() Manager {
	switch {
	case have("pacman"):
		return Pacman
	case have("apt-get"):
		return APT
	case have("dnf"):
		return Dnf
	}
	return Unknown
}

// applyPackages fills in the platform-tools package names for the manager.
func (info *Info) applyPackages() {
	info.Packages = nil
	info.UdevPackages = nil
	info.Alternates = nil

	switch info.Manager {
	case Pacman:
		// Arch/Artix: the android-tools package ships /usr/bin/adb and
		// /usr/bin/fastboot. android-udev carries the udev rules.
		info.Packages = []string{"android-tools"}
		info.UdevPackages = []string{"android-udev"}
		info.Alternates = []string{"android-platform-tools"}
	case APT:
		// Debian/Ubuntu/Mint ship adb and fastboot directly. The platform-tools
		// common package supplies the udev rules so no root is needed.
		info.Packages = []string{"adb", "fastboot", "android-sdk-platform-tools-common"}
		info.Alternates = []string{
			"android-tools-adb android-tools-fastboot",
			"google-android-platform-tools-installer",
		}
	case Dnf:
		info.Packages = []string{"android-tools"}
		info.Alternates = []string{"platform-tools"}
	}
}

func readOSRelease() (id string, like []string, pretty string) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "", nil, ""
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		switch key {
		case "ID":
			id = value
		case "ID_LIKE":
			like = strings.Fields(value)
		case "PRETTY_NAME":
			pretty = value
		}
	}
	return id, like, pretty
}

func have(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// Have reports whether an executable is present in PATH.
func Have(bin string) bool { return have(bin) }

// Which resolves a binary to its absolute path, or "" when it is absent.
func Which(bin string) string {
	p, err := exec.LookPath(bin)
	if err != nil {
		return ""
	}
	return p
}

// MissingTools returns the subset of tools that is not installed.
func MissingTools(tools ...string) []string {
	var out []string
	for _, t := range tools {
		if !have(t) {
			out = append(out, t)
		}
	}
	return out
}

// Home returns the user home directory, falling back to "/" if unset.
func Home() string {
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return h
	}
	return "/"
}

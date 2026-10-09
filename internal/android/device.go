// Package android talks to adb and fastboot and models what it finds.
package android

import (
	"context"
	"strings"
	"time"

	"adm/internal/runner"
)

// Mode tells which tool the device is currently reachable through.
type Mode string

const (
	ModeADB      Mode = "adb"
	ModeFastboot Mode = "fastboot"
)

// BootState is the bootloader lock status.
type BootState int

const (
	BootUnknown BootState = iota
	BootLocked
	BootUnlocked
)

func (b BootState) String() string {
	switch b {
	case BootLocked:
		return "locked"
	case BootUnlocked:
		return "unlocked"
	default:
		return "unknown"
	}
}

// Device is one reachable phone.
type Device struct {
	Serial  string
	Mode    Mode
	State   string
	Model   string
	Product string
	// Codename is the ro.product.device value, e.g. "raven".
	Codename string
	// Transport is "usb" or "tcp".
	Transport string
	Boot      BootState
	// Reason explains an offline or unauthorized state when adb gave one.
	Reason string
}

// Label is a short human readable device name used in lists.
func (d Device) Label() string {
	switch {
	case d.Model != "" && d.Codename != "":
		return d.Model + " (" + d.Codename + ")"
	case d.Model != "":
		return d.Model
	case d.Codename != "":
		return d.Codename
	default:
		return d.Serial
	}
}

// TCP reports whether the device was reached over the network.
func (d Device) TCP() bool { return d.Transport == "tcp" }

// ADB builds a complete adb argument vector addressing this device. The
// program name is included so the result can be handed straight to exec.
func (d Device) ADB(args ...string) []string {
	out := []string{"adb", "-s", d.Serial}
	return append(out, args...)
}

// Fastboot builds a complete fastboot argument vector addressing this device.
func (d Device) Fastboot(args ...string) []string {
	out := []string{"fastboot", "-s", d.Serial}
	return append(out, args...)
}

// Scans adb and fastboot and returns every reachable device. adb results come
// first so a phone in normal mode is listed before one sitting in fastboot.
func Scan(ctx context.Context) ([]Device, error) {
	var out []Device

	adbDevices, adbErr := scanADB(ctx)
	out = append(out, adbDevices...)

	fbDevices, fbErr := scanFastboot(ctx)
	out = append(out, fbDevices...)

	if adbErr != nil && fbErr != nil {
		return out, adbErr
	}
	return out, nil
}

func scanADB(ctx context.Context) ([]Device, error) {
	res := runner.RunQuiet(ctx, "adb", "devices", "-l")
	if res.Err != nil {
		return nil, res.Err
	}
	if res.Code != 0 {
		return nil, runner.ErrExit{Command: "adb devices -l", Code: res.Code}
	}
	return parseADBDevices(res.Out), nil
}

// parseADBDevices turns the output of "adb devices -l" into device records.
func parseADBDevices(out string) []Device {
	var devs []Device
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "List of devices") ||
			strings.HasPrefix(line, "*") {
			continue
		}

		// Serial, state and the optional key:value extras are separated by
		// tabs or runs of spaces.
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		d := Device{
			Serial:    fields[0],
			Mode:      ModeADB,
			State:     fields[1],
			Transport: transportOf(fields[0]),
		}
		for _, extra := range fields[2:] {
			key, value, ok := strings.Cut(extra, ":")
			if !ok {
				continue
			}
			switch key {
			case "model":
				d.Model = value
			case "product":
				d.Product = value
			case "device":
				if d.Codename == "" {
					d.Codename = value
				}
			case "unauthorized":
				d.Reason = value
			}
		}
		devs = append(devs, d)
	}
	return devs
}

// transportOf guesses the transport from the serial shape. Emulators and
// network devices always carry a colon or the emulator- prefix.
func transportOf(serial string) string {
	if strings.HasPrefix(serial, "emulator-") || strings.Contains(serial, ":") {
		return "tcp"
	}
	return "usb"
}

func scanFastboot(ctx context.Context) ([]Device, error) {
	res := runner.RunQuiet(ctx, "fastboot", "devices")
	if res.Err != nil {
		return nil, res.Err
	}
	if res.Code != 0 {
		return nil, runner.ErrExit{Command: "fastboot devices", Code: res.Code}
	}
	return parseFastbootDevices(res.Out), nil
}

// parseFastbootDevices turns the output of "fastboot devices" into records.
func parseFastbootDevices(out string) []Device {
	var devs []Device
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}
		// fastboot devices prints "<serial>\tfastboot" but some builds pad
		// with spaces, so treat any occurrence of the keyword the same way.
		idx := strings.Index(strings.ToLower(line), "fastboot")
		if idx < 0 {
			continue
		}
		serial := strings.TrimSpace(line[:idx])
		if serial == "" {
			continue
		}
		devs = append(devs, Device{
			Serial:    serial,
			Mode:      ModeFastboot,
			State:     "fastboot",
			Transport: transportOf(serial),
		})
	}
	return devs
}

// Enrich fills in codename, model and bootloader state for the given device.
// It is best effort: any field it cannot read is left empty or unknown.
func Enrich(ctx context.Context, d *Device) {
	if d == nil {
		return
	}

	// A TCP device is queried over the network and may be slow, so give it
	// more room than the default quiet timeout.
	qctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	switch d.Mode {
	case ModeFastboot:
		enrichFastboot(qctx, d)
	case ModeADB:
		enrichADB(qctx, d)
	}
}

func enrichADB(ctx context.Context, d *Device) {
	if d.State != "device" {
		// Only a fully authorised device answers getprop.
		return
	}

	d.Codename = firstProp(ctx, d.Serial, "ro.product.device", "ro.build.product", "ro.product.vendor.device")
	d.Model = firstProp(ctx, d.Serial, "ro.product.model", "ro.product.vendor.model")
	d.Product = firstProp(ctx, d.Serial, "ro.build.product")

	d.Boot = bootStateFromProps(ctx, d.Serial)
}

func firstProp(ctx context.Context, serial string, keys ...string) string {
	for _, key := range keys {
		res := runner.RunQuiet(ctx, "adb", "-s", serial, "shell", "getprop", key)
		if res.Err != nil {
			continue
		}
		v := strings.TrimSpace(res.Out)
		if v != "" {
			return v
		}
	}
	return ""
}

// bootStateFromProps derives the bootloader lock status from the properties
// Android exposes to an ordinary shell user.
func bootStateFromProps(ctx context.Context, serial string) BootState {
	state := BootUnknown

	if locked := firstProp(ctx, serial, "ro.boot.flash.locked"); locked != "" {
		switch strings.TrimSpace(locked) {
		case "1":
			state = BootLocked
		case "0":
			state = BootUnlocked
		}
	}

	// vbmeta.device_state is only present on devices with a modern AVB setup.
	if vs := firstProp(ctx, serial, "ro.boot.vbmeta.device_state"); vs != "" {
		switch strings.ToLower(strings.TrimSpace(vs)) {
		case "locked":
			state = BootLocked
		case "unlocked":
			state = BootUnlocked
		}
	}

	// verifiedbootstate is the last resort and also catches relock requests.
	if vbs := firstProp(ctx, serial, "ro.boot.verifiedbootstate"); vbs != "" {
		switch strings.ToLower(strings.TrimSpace(vbs)) {
		case "green":
			// Only trust green when nothing else claimed a state, since a
			// relock may still be pending.
			if state == BootUnknown {
				state = BootLocked
			}
		case "orange":
			state = BootUnlocked
		case "yellow", "red":
			state = BootUnlocked
		}
	}

	return state
}

func enrichFastboot(ctx context.Context, d *Device) {
	// fastboot exposes the OEM lock state through getvar. Output lands on
	// stderr with a "name: value" shape, which runner merges into Out.
	vars := getvarAll(ctx, d.Serial, "unlocked", "is-userspace", "secure", "current-unlocked")

	for _, v := range vars {
		name, value, ok := strings.Cut(v, ":")
		if !ok {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.ToLower(strings.TrimSpace(value))
		switch name {
		case "unlocked":
			switch value {
			case "yes":
				d.Boot = BootUnlocked
			case "no":
				d.Boot = BootLocked
			}
		case "is-userspace":
			if value == "no" && d.Boot == BootUnknown {
				// A userspace fastboot still enforces the bootloader lock.
				d.Boot = BootLocked
			}
		}
	}

	if v := getvarOne(ctx, d.Serial, "product"); v != "" {
		d.Codename = v
	}
	if v := getvarOne(ctx, d.Serial, "variant"); v != "" {
		d.Product = v
	}
}

func getvarOne(ctx context.Context, serial, name string) string {
	res := runner.RunQuiet(ctx, "fastboot", "-s", serial, "getvar", name)
	if res.Err != nil {
		return ""
	}
	return parseGetvar(res.Out, name)
}

func getvarAll(ctx context.Context, serial string, names ...string) []string {
	var out []string
	for _, n := range names {
		res := runner.RunQuiet(ctx, "fastboot", "-s", serial, "getvar", n)
		if res.Err != nil {
			continue
		}
		if v := parseGetvar(res.Out, n); v != "" {
			out = append(out, n+":"+v)
		}
	}
	return out
}

func parseGetvar(out, name string) string {
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(key), name) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// StartServer starts the adb daemon, returning its output.
func StartServer(ctx context.Context) runner.Result {
	return runner.Run(ctx, "adb", "start-server")
}

// ConnectTCP connects to a device over the network for both adb and fastboot.
func ConnectTCP(ctx context.Context, host string) []runner.Result {
	var out []runner.Result
	out = append(out, runner.RunQuiet(ctx, "adb", "connect", host))
	out = append(out, runner.RunQuiet(ctx, "fastboot", "connect", host))
	return out
}

// DisconnectTCP drops every network device.
func DisconnectTCP(ctx context.Context) []runner.Result {
	var out []runner.Result
	out = append(out, runner.RunQuiet(ctx, "adb", "disconnect"))
	out = append(out, runner.RunQuiet(ctx, "fastboot", "disconnect"))
	return out
}

// Find looks a device up by serial in a scan result.
func Find(devices []Device, serial string) (Device, bool) {
	for _, d := range devices {
		if d.Serial == serial {
			return d, true
		}
	}
	return Device{}, false
}

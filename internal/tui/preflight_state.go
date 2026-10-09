package tui

import "adm/internal/sysinfo"

// preflightPhase tracks where the environment check has got to.
type preflightPhase int

const (
	phaseChecking preflightPhase = iota
	phaseReady
	phaseMissing
	phaseInstalling
	phaseInstalled
	phaseFailed
	phaseManual
	phaseDeclined
)

// preflight holds the state of the adb/fastboot availability check and the
// optional install of platform-tools.
type preflight struct {
	phase    preflightPhase
	adb      string
	fastboot string
	missing  []string
	info     sysinfo.Info
	plan     sysinfo.Plan
	idx      int
	manual   string
	manualOK bool
	fatal    bool
}

package sysinfo

import "testing"

func TestDetectOnThisMachine(t *testing.T) {
	info := Detect()
	t.Logf("id=%q like=%v manager=%q pretty=%q", info.ID, info.Like, info.Manager, info.Pretty)
	if info.Manager == Unknown {
		t.Skip("unrecognised distribution, nothing to assert")
	}
	if !info.Known() {
		t.Errorf("Detect found manager %q but Known() is false", info.Manager)
	}
}

func TestSupportedDistros(t *testing.T) {
	// Each supported target must resolve to the right package manager and
	// package names. This is the table the install flow depends on.
	cases := []struct {
		id, wantPkg, altHint string
		like                 []string
		want                 Manager
	}{
		{id: "arch", want: Pacman, wantPkg: "android-tools", altHint: "android-platform-tools"},
		{id: "artix", want: Pacman, wantPkg: "android-tools", altHint: "android-platform-tools"},
		{id: "ubuntu", want: APT, wantPkg: "adb", altHint: "android-tools-adb"},
		{id: "debian", want: APT, wantPkg: "adb", altHint: "android-tools-adb"},
		{id: "mint", want: APT, wantPkg: "adb", altHint: "android-tools-adb"},
		{id: "", like: []string{"arch", "manjaro"}, want: Pacman, wantPkg: "android-tools", altHint: "android-platform-tools"},
		{id: "", like: []string{"debian"}, want: APT, wantPkg: "adb", altHint: "android-tools-adb"},
	}

	for _, c := range cases {
		info := classify(c.id, c.like)
		if info.Manager != c.want {
			t.Errorf("id=%q like=%v: manager = %q, want %q", c.id, c.like, info.Manager, c.want)
			continue
		}
		if !containsSubstring(info.Packages, c.wantPkg) {
			t.Errorf("id=%q: packages %v do not include %q", c.id, info.Packages, c.wantPkg)
		}
		if !containsSubstring(info.Alternates, c.altHint) {
			t.Errorf("id=%q: alternates %v do not include %q", c.id, info.Alternates, c.altHint)
		}
		if !info.Supported() {
			t.Errorf("id=%q: not reported as supported", c.id)
		}
	}
}

func TestUnknownDistribution(t *testing.T) {
	info := classify("plan9", nil)
	if info.Known() {
		t.Errorf("plan9 reported as known: %+v", info)
	}
	if info.InstallCommand().Command != "" {
		t.Errorf("unknown distro produced an install command: %q", info.InstallCommand().Command)
	}
}

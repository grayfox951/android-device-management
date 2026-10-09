package sysinfo

import "testing"

func TestInstallCommands(t *testing.T) {
	arch := classify("arch", nil)
	plan := arch.InstallCommand()
	if plan.Command != "sudo pacman -S --needed --noconfirm android-tools android-udev" {
		t.Errorf("arch install command = %q", plan.Command)
	}

	deb := classify("ubuntu", nil)
	plan = deb.InstallCommand()
	if plan.Command != "sudo apt-get install -y adb fastboot android-sdk-platform-tools-common" {
		t.Errorf("ubuntu install command = %q", plan.Command)
	}
}

func TestMissingTools(t *testing.T) {
	// "sh" always exists on the supported systems, the fake name never does.
	if got := MissingTools("sh", "definitely-not-real-xyz"); len(got) != 1 || got[0] != "definitely-not-real-xyz" {
		t.Errorf("MissingTools = %#v", got)
	}
	if got := MissingTools("sh"); len(got) != 0 {
		t.Errorf("MissingTools(sh) = %#v", got)
	}
}

func TestHomeIsAbsolute(t *testing.T) {
	if h := Home(); h == "" || h[0] != '/' {
		t.Errorf("Home() = %q, want an absolute path", h)
	}
}

func containsSubstring(list []string, want string) bool {
	for _, s := range list {
		if s != "" && contains(s, want) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

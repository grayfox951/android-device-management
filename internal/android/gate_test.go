package android

import (
	"testing"
)

func TestLockedGateHidesRiskyCommands(t *testing.T) {
	// A locked bootloader must not expose anything that flashes or erases.
	for _, c := range Catalog(ModeFastboot) {
		if c.Category != CatFlash {
			continue
		}
		if c.AvailableOn(BootLocked) {
			t.Errorf("%s (%s) is offered on a locked bootloader", c.ID, c.Label)
		}
		if !c.AvailableOn(BootUnlocked) {
			t.Errorf("%s (%s) is hidden on an unlocked bootloader", c.ID, c.Label)
		}
	}
}

func TestUnlockWorksWhileLocked(t *testing.T) {
	// The whole point of a locked bootloader is that unlocking is available.
	var found bool
	for _, c := range Catalog(ModeFastboot) {
		if c.ID == "fb.unlock" {
			found = true
			if !c.AvailableOn(BootLocked) {
				t.Error("fastboot flashing unlock is hidden on a locked bootloader")
			}
		}
	}
	if !found {
		t.Fatal("no fb.unlock command in the fastboot catalog")
	}
}

func TestRootCommandsNeedUnlocked(t *testing.T) {
	for _, c := range Catalog(ModeADB) {
		if c.Root && c.AvailableOn(BootLocked) {
			t.Errorf("%s requires root but is offered on a locked bootloader", c.ID)
		}
	}
}

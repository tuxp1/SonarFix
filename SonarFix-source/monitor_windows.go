//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"
)

// This scheduler wraps the existing guarded audio operations; it never mutes
// or disables an endpoint itself. A probe failure always takes the full path.
func discordProcessPresent() (bool, error) {
	h, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return true, err
	}
	defer syscall.CloseHandle(h)
	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err = syscall.Process32First(h, &entry); err != nil {
		return true, err
	}
	for {
		if strings.EqualFold(syscall.UTF16ToString(entry.ExeFile[:]), "Discord.exe") {
			return true, nil
		}
		err = syscall.Process32Next(h, &entry)
		if err == syscall.ERROR_NO_MORE_FILES {
			return false, nil
		}
		if err != nil {
			return true, err
		}
	}
}

func canIdleEcho(p preferences, discord bool) bool {
	return p.Echo && !discord && len(p.EchoMutes) == 0
}
func noMonitoringWork(p preferences) bool {
	return !p.Echo && !p.DisableBoth && len(p.EchoMutes)+len(p.Changed)+len(p.Pending)+len(p.Defaults) == 0
}
func monitoringDelay(p preferences, discord bool) uint32 {
	if !p.DisableBoth && !discord {
		return 6000
	}
	return 3000
}
func outputsAlreadyDisabled(ds []renderEndpoint, defaults []string, pending int) bool {
	if pending > 0 || len(defaults) != 3 {
		return false
	}
	targets := map[string]bool{}
	game, chat := false, false
	for _, d := range ds {
		if validTarget(d) {
			if d.State != deviceStateDisabled {
				return false
			}
			targets[d.ID] = true
			name := canonicalSonarOutput(d.Name)
			if strings.HasSuffix(name, " - chat") {
				chat = true
			} else {
				game = true
			}
		}
	}
	if !game || !chat {
		return false
	}
	for _, id := range defaults {
		if id == "" || targets[id] {
			return false
		}
	}
	return true
}
func gameChatNeedsApply(p preferences) bool {
	if len(p.Pending) > 0 || validateOutput(p.OutputID) != nil {
		return true
	}
	en, err := createEnumerator()
	if err != nil {
		return true
	}
	defer release(en)
	ds, err := listEndpoints(en, 0, 3)
	if err != nil {
		return true
	}
	return !outputsAlreadyDisabled(ds, []string{defaultID(en, 0), defaultID(en, 1), defaultID(en, 2)}, len(p.Pending))
}

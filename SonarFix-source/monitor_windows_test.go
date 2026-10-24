//go:build windows

package main

import (
	"testing"
)

func TestMonitorIdlePreservesOwnership(t *testing.T) {
	p := preferences{}
	p.Echo = true
	if !canIdleEcho(p, false) || canIdleEcho(p, true) {
		t.Fatal("Discord presence not respected")
	}
	p.EchoMutes = make([]ownedRenderMute, 1)
	if canIdleEcho(p, false) {
		t.Fatal("owned mute restoration could be skipped")
	}
	if noMonitoringWork(p) {
		t.Fatal("monitor exited with recovery work")
	}
	p = preferences{}
	if !noMonitoringWork(p) {
		t.Fatal("unneeded background process retained")
	}
	p.Pending = []string{"pending"}
	if noMonitoringWork(p) {
		t.Fatal("pending recovery lost")
	}
}
func TestMonitorDelay(t *testing.T) {
	p := preferences{}
	p.Echo = true
	if monitoringDelay(p, false) != 6000 || monitoringDelay(p, true) != 3000 {
		t.Fatal("idle/active delay")
	}
	p.DisableBoth = true
	if monitoringDelay(p, false) != 3000 {
		t.Fatal("Game/Chat enforcement delayed")
	}
}
func TestMonitorReappliesChangedEndpoints(t *testing.T) {
	ds := []renderEndpoint{
		{ID: "{0.0.0.game}", Name: "SteelSeries Sonar - Gaming", Identity: "SteelSeries", State: 2, Flow: 0},
		{ID: "{0.0.0.chat}", Name: "SteelSeries Sonar - Chat", Identity: "SteelSeries", State: 2, Flow: 0},
	}
	defaults := []string{"physical", "physical", "physical"}
	if !outputsAlreadyDisabled(ds, defaults, 0) {
		t.Fatal("stable state cannot skip reapplying")
	}
	ds[0].State = 1
	if outputsAlreadyDisabled(ds, defaults, 0) {
		t.Fatal("Sonar re-enable was missed")
	}
	ds[0].State = 2
	if outputsAlreadyDisabled(ds, defaults, 1) {
		t.Fatal("pending journal was skipped")
	}
	if outputsAlreadyDisabled(ds, []string{ds[0].ID, "physical", "physical"}, 0) {
		t.Fatal("unsafe default was skipped")
	}
	if outputsAlreadyDisabled(ds[:1], defaults, 0) || outputsAlreadyDisabled(nil, defaults, 0) {
		t.Fatal("missing endpoint assumed disabled")
	}
	ds[1].Name = "SteelSeries Sonar - Gaming"
	if outputsAlreadyDisabled(ds, defaults, 0) {
		t.Fatal("missing Chat assumed ready")
	}
	ds[1].Name = "SteelSeries Sonar - Chat"
	ds[1].Flow = 1
	if outputsAlreadyDisabled(ds, defaults, 0) {
		t.Fatal("capture mistaken for a target")
	}
}
func BenchmarkDiscordProcessProbe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := discordProcessPresent(); err != nil {
			b.Fatal(err)
		}
	}
}

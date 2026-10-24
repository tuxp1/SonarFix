//go:build windows

package main

import (
	"reflect"
	"testing"
)

func healthFixture() (preferences, []renderEndpoint) {
	p := preferences{}
	p.OutputID = "{0.0.0.usb}"
	id := "SteelSeries Sonar Virtual Audio Device"
	return p, []renderEndpoint{
		{ID: "{0.0.0.game}", Name: "SteelSeries Sonar - Gaming", Identity: id, Flow: 0, State: 1},
		{ID: "{0.0.0.chat}", Name: "SteelSeries Sonar - Chat", Identity: id, Flow: 0, State: 1},
		{ID: "{0.0.1.mic}", Name: "SteelSeries Sonar - Microphone", Identity: id, Flow: 1, State: 1},
		{ID: p.OutputID, Name: "Physical Headphones", Identity: `USB\VID_TEST`, Flow: 0, State: 1},
	}
}
func TestDeviceHealthStates(t *testing.T) {
	p, ds := healthFixture()
	if evaluateHealth(p, ds).Status != "ready" {
		t.Fatal("healthy active devices")
	}
	p.DisableBoth = true
	if evaluateHealth(p, ds).Status != "action" {
		t.Fatal("re-enabled Game/Chat missed")
	}
	ds[0].State = 2
	ds[1].State = 2
	if evaluateHealth(p, ds).Status != "ready" {
		t.Fatal("disabled devices should match preference")
	}
	for _, index := range []int{2, 3} {
		before := ds[index].State
		ds[index].State = 2
		if evaluateHealth(p, ds).Status != "unavailable" {
			t.Fatal("missing microphone/output reported ready")
		}
		ds[index].State = before
	}
	if evaluateHealth(p, ds[:1]).Status != "unavailable" {
		t.Fatal("missing Sonar endpoints reported ready")
	}
	p.OutputID = "unknown"
	if evaluateHealth(p, ds).Status != "unavailable" {
		t.Fatal("missing chosen output")
	}
	p, ds = healthFixture()
	ds[0].Identity = "Unknown driver"
	if evaluateHealth(p, ds).Status != "unavailable" {
		t.Fatal("unverified endpoint accepted")
	}
	p, ds = healthFixture()
	p.Pending = []string{"pending"}
	if evaluateHealth(p, ds).Status != "action" {
		t.Fatal("pending transaction ignored")
	}
}
func TestDeviceHealthIsReadOnly(t *testing.T) {
	p, ds := healthFixture()
	before := append([]renderEndpoint(nil), ds...)
	evaluateHealth(p, ds)
	if !reflect.DeepEqual(before, ds) {
		t.Fatal("health changed device snapshot")
	}
	p.DisableBoth = true
	ds[0].State = 2
	ds[1].State = 2
	duplicate := ds[0]
	duplicate.State = 1
	ds = append(ds, duplicate)
	if evaluateHealth(p, ds).Status != "action" {
		t.Fatal("mixed duplicate endpoints reported ready")
	}
}
func TestOptionalTrayDefaults(t *testing.T) {
	for _, raw := range []string{"", `{}`, `null`, `{"show_tray":false}`, `{"show_tray":"true"}`, `broken`} {
		if trayRequested([]byte(raw)) {
			t.Fatalf("tray enabled unexpectedly: %q", raw)
		}
	}
	if !trayRequested([]byte(`{"show_tray":true,"automatic_updates":false}`)) {
		t.Fatal("saved opt-in was ignored")
	}
}

//go:build windows

package main

import "strings"

// Health checks are read-only and intentionally do not enumerate audio sessions.
type healthReport struct {
	Status, Gaming, Chat, Microphone, Output string
	Sonar, DisableExpected, WorkerRunning    bool
}

func sonarHealthEndpoint(d renderEndpoint) bool {
	return strings.Contains(strings.ToLower(d.Identity), "steelseries sonar virtual audio device") && strings.Contains(strings.ToLower(d.Name), "sonar")
}
func healthState(ds []renderEndpoint) string {
	if len(ds) == 0 {
		return "unavailable"
	}
	active, disabled := 0, 0
	for _, d := range ds {
		if d.State == 1 {
			active++
		}
		if d.State == 2 {
			disabled++
		}
	}
	if active == len(ds) {
		return "active"
	}
	if disabled == len(ds) {
		return "disabled"
	}
	if active > 0 || disabled > 0 {
		return "mixed"
	}
	return "unavailable"
}
func evaluateHealth(p preferences, ds []renderEndpoint) healthReport {
	r := healthReport{Status: "unavailable", DisableExpected: p.DisableBoth, Microphone: "unavailable", Output: "unavailable"}
	games, chats := []renderEndpoint{}, []renderEndpoint{}
	for _, d := range ds {
		if d.ID == p.OutputID && physicalOutput(d) {
			r.Output = "active"
		}
		if !sonarHealthEndpoint(d) {
			continue
		}
		r.Sonar = true
		if d.Flow == 1 && sonarMic(d.Name) && d.State == 1 {
			r.Microphone = "active"
		}
		if !validTarget(d) {
			continue
		}
		if strings.HasSuffix(canonicalSonarOutput(d.Name), " - chat") {
			chats = append(chats, d)
		} else {
			games = append(games, d)
		}
	}
	r.Gaming = healthState(games)
	r.Chat = healthState(chats)
	if !r.Sonar || r.Gaming == "unavailable" || r.Chat == "unavailable" || r.Microphone != "active" || r.Output != "active" {
		return r
	}
	expected := "active"
	if p.DisableBoth {
		expected = "disabled"
	}
	r.Status = "ready"
	if r.Gaming != expected || r.Chat != expected || len(p.Pending) > 0 || (!p.DisableBoth && len(p.Changed)+len(p.Defaults) > 0) {
		r.Status = "action"
	}
	return r
}
func inspectDeviceHealth(p preferences) (healthReport, error) {
	en, e := createEnumerator()
	if e != nil {
		return healthReport{}, e
	}
	defer release(en)
	ds := []renderEndpoint{}
	for flow := uintptr(0); flow < 2; flow++ {
		items, err := listEndpoints(en, flow, 15)
		if err != nil {
			return healthReport{}, err
		}
		ds = append(ds, items...)
	}
	r := evaluateHealth(p, ds)
	r.WorkerRunning = isRunning()
	return r, nil
}
func writeHealthInfo(path string) error {
	p, e := loadPreferences()
	if e != nil {
		return e
	}
	r, e := inspectDeviceHealth(p)
	if e != nil {
		return e
	}
	return atomicJSON(path, r)
}

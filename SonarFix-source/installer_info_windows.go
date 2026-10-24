//go:build windows

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf16"
)

// sonarMicrophonePresent reports whether both active Sonar Microphone
// endpoints (playback and recording) exist. Read-only, for status display.
func sonarMicrophonePresent(endpoints []endpointFacts) bool {
	render, capture := false, false
	for _, e := range endpoints {
		render = render || sonarMicEndpoint(e, 0, "{0.0.0.")
		capture = capture || sonarMicEndpoint(e, 1, "{0.0.1.")
	}
	return render && capture
}

func writeSetupInfo(path string) error {
	p, e := loadPreferences()
	if e != nil {
		return e
	}
	// New installs default to the echo fix: the route is detected automatically.
	if name, e := settingsPath(); e == nil {
		if _, e = os.Stat(name); os.IsNotExist(e) {
			p.Echo = true
		}
	}
	en, e := createEnumerator()
	if e != nil {
		return e
	}
	defer release(en)
	ds, e := listEndpoints(en, 0, 1)
	if e != nil {
		return e
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[Settings]\r\nEcho=%t\r\nDisable=%t\r\nStartup=%t\r\nRedirect=%t\r\nLanguage=%s\r\nOutput=%s\r\n", p.Echo, p.DisableBoth, p.Startup && !startupDisabledByUser(), p.AllowRedirect, p.Language, p.OutputID)
	fmt.Fprintf(&b, "Running=%t\r\n", isRunning())
	fmt.Fprintf(&b, "EchoVerified=%t\r\n", p.EchoRoute != nil)
	active := false
	echoState := "waiting"
	if !p.Echo {
		echoState = "off"
	}
	if p.Echo {
		endpoints, defaults, err := collectEndpointFacts()
		switch {
		case err != nil:
			echoState = "unknown"
		case !sonarMicrophonePresent(endpoints):
			// Without the Sonar Microphone there is nothing to wait for.
			echoState = "device_unavailable"
		case p.EchoRoute == nil:
			_, reason := discoverEchoRoute(endpoints, defaults, "ui-inspection-only")
			if reason != "no_discord_playback_on_sonar_microphone" {
				echoState = "ambiguous"
			}
		}
	}
	if p.Echo && p.EchoRoute != nil && echoState == "waiting" {
		if s, e := inspectEcho(p.EchoRoute); e == nil {
			ok, _ := confirmedEchoDecision(p.EchoRoute, snapshotFacts(s))
			active = ok && s.Render.Muted
			if ok {
				echoState = "ready"
				if active {
					echoState = "blocked"
				}
			} else {
				echoState = "ambiguous"
			}
		}
	}
	fmt.Fprintf(&b, "EchoActive=%t\r\n", active)
	// UI inspection only; no route or device mutations.
	fmt.Fprintf(&b, "EchoState=%s\r\n", echoState)
	pending := len(p.Changed)+len(p.Pending)+len(p.Defaults)+len(p.EchoMutes) > 0
	fmt.Fprintf(&b, "RestorePending=%t\r\n", pending)
	deviceStatus := "off"
	if !p.DisableBoth && (len(p.Changed)+len(p.Pending)+len(p.Defaults) > 0) {
		deviceStatus = "restore_pending"
	}
	if p.DisableBoth {
		deviceStatus = "output_needed"
		if validateOutput(p.OutputID) == nil {
			deviceStatus = "pending"
			targets, disabled := 0, 0
			if all, err := listEndpoints(en, 0, 3); err == nil {
				for _, d := range all {
					if validTarget(d) {
						targets++
						if endpointState(en, d.ID) == deviceStateDisabled {
							disabled++
						}
					}
				}
			}
			if targets >= 2 && targets == disabled && len(p.Pending) == 0 {
				deviceStatus = "disabled"
			}
		}
	}
	fmt.Fprintf(&b, "DeviceState=%s\r\n", deviceStatus)
	i := 0
	for _, d := range ds {
		if physicalOutput(d) {
			fmt.Fprintf(&b, "[Output%d]\r\nID=%s\r\nName=%s\r\n", i, d.ID, strings.NewReplacer("\r", " ", "\n", " ").Replace(d.Name))
			i++
		}
	}
	fmt.Fprintf(&b, "[Outputs]\r\nCount=%s\r\nDefault=%s\r\n", strconv.Itoa(i), defaultID(en, 0))
	// GetPrivateProfileString (Inno) recognizes UTF-16 with a BOM.
	u := utf16.Encode([]rune(b.String()))
	raw := []byte{255, 254}
	for _, v := range u {
		raw = append(raw, byte(v), byte(v>>8))
	}
	return os.WriteFile(path, raw, 0600)
}

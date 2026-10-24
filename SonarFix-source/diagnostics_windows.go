//go:build windows

package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unsafe"
)

var iidMMEndpoint = guid{0x1be09788, 0x6894, 0x4089, [8]byte{0x85, 0x86, 0x9a, 0x2a, 0x6c, 0x26, 0x5a, 0xc5}}
var iidAudioMeter = guid{0xc02216f6, 0x8c67, 0x4b5b, [8]byte{0x9d, 0, 0xd0, 8, 0xe7, 0x3e, 0, 0x64}}
var iidEndpointVolume = guid{0x5cdf2c82, 0x841e, 0x4546, [8]byte{0x97, 0x22, 0x0c, 0xf7, 0x40, 0x78, 0x22, 0x9a}}
var identityKey = propertyKey{guid{0xb3f8fa53, 0x0004, 0x438e, [8]byte{0x90, 0x03, 0x51, 0xa4, 0x6e, 0x13, 0x9b, 0xfc}}, 2}
var interfaceKey = propertyKey{identityKey.fmtid, 6}

func deviceProperty(d uintptr, key propertyKey) string {
	var s uintptr
	if !hrOK(comCall(d, 4, 0, uintptr(unsafe.Pointer(&s)))) || s == 0 {
		return ""
	}
	defer release(s)
	var pv [3]uintptr
	if !hrOK(comCall(s, 5, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&pv[0])))) {
		return ""
	}
	defer procPropVariantClear.Call(uintptr(unsafe.Pointer(&pv[0])))
	if *(*uint16)(unsafe.Pointer(&pv[0])) != 31 {
		return ""
	}
	return utf16z((*uint16)(unsafe.Pointer(pv[1])))
}
func deviceIdentity(d uintptr) string {
	return deviceProperty(d, identityKey) + " | " + deviceProperty(d, interfaceKey)
}
func endpointFlow(d uintptr) int32 {
	p := queryInterface(d, &iidMMEndpoint)
	if p == 0 {
		return -1
	}
	defer release(p)
	var f int32 = -1
	if !hrOK(comCall(p, 3, uintptr(unsafe.Pointer(&f)))) {
		return -1
	}
	return f
}

type sessionReport struct {
	Endpoint, Instance, Executable, Reason string
	ExecutablePath                         string
	PID                                    uint32
	Flow, State                            int32
	Muted                                  bool
	Volume                                 float32
	Peak                                   float32
}
type endpointReport struct {
	renderEndpoint
	Peak           float32
	Sessions       []sessionReport
	Error          string
	EndpointMuted  bool
	EndpointVolume float32
}
type diagnosticReport struct {
	Version       string
	Time          string
	SessionMuting bool
	// Session Reason values show the generic unconfirmed policy. EchoDecision
	// is the confirmed-route verdict the worker actually acts on.
	EchoDecision  string
	WorkerRunning bool
	Defaults      map[uint32]string
	Endpoints     []endpointReport
}

func eachSession(d uintptr, visit func(uintptr, sessionReport) error) error {
	var manager uintptr
	h := comCall(d, 3, uintptr(unsafe.Pointer(&iidAudioSessionManager2)), 1, 0, uintptr(unsafe.Pointer(&manager)))
	if !hrOK(h) || manager == 0 {
		return fmt.Errorf("session manager: %08x", uint32(h))
	}
	defer release(manager)
	var sessions uintptr
	if !hrOK(comCall(manager, 5, uintptr(unsafe.Pointer(&sessions)))) || sessions == 0 {
		return errors.New("session enumeration failed")
	}
	defer release(sessions)
	var n int32
	if !hrOK(comCall(sessions, 3, uintptr(unsafe.Pointer(&n)))) {
		return errors.New("session count failed")
	}
	for i := int32(0); i < n; i++ {
		var ctrl uintptr
		if !hrOK(comCall(sessions, 4, uintptr(i), uintptr(unsafe.Pointer(&ctrl)))) || ctrl == 0 {
			continue
		}
		err := func() error {
			defer release(ctrl)
			c := queryInterface(ctrl, &iidAudioSessionControl2)
			if c == 0 {
				return nil
			}
			defer release(c)
			var pid uint32
			var state int32 = -1
			// S_NO_SINGLE_PROCESS is not sufficient process identity.
			if comCall(c, 14, uintptr(unsafe.Pointer(&pid))) != 0 {
				return nil
			}
			if !hrOK(comCall(c, 3, uintptr(unsafe.Pointer(&state)))) {
				return nil
			}
			s := sessionReport{Endpoint: deviceID(d), Instance: getSessionInstance(c), Executable: executableName(pid), PID: pid, Flow: endpointFlow(d), State: state}
			s.ExecutablePath = executablePath(pid)
			vol := queryInterface(ctrl, &iidSimpleAudioVolume)
			if vol == 0 {
				return nil
			}
			defer release(vol)
			var muted int32
			if !hrOK(comCall(vol, 6, uintptr(unsafe.Pointer(&muted)))) {
				return nil
			}
			s.Muted = muted != 0
			meter := queryInterface(ctrl, &iidAudioMeter)
			if meter != 0 {
				comCall(meter, 3, uintptr(unsafe.Pointer(&s.Peak)))
				release(meter)
			}
			if !hrOK(comCall(vol, 4, uintptr(unsafe.Pointer(&s.Volume)))) {
				return nil
			}
			_, s.Reason = muteDecision(sessionFacts{s.Endpoint, s.Instance, s.Executable, s.Flow, s.State, s.PID})
			return visit(vol, s)
		}()
		if err != nil {
			return err
		}
	}
	return nil
}
func writeDiagnostics(path string) error {
	en, e := createEnumerator()
	if e != nil {
		return e
	}
	defer release(en)
	p, _ := loadPreferences()
	r := diagnosticReport{Version: appVersion, Time: time.Now().Format(time.RFC3339), SessionMuting: p.Echo && p.EchoRoute != nil, WorkerRunning: isRunning(), Defaults: map[uint32]string{}}
	for role := uint32(0); role < 3; role++ {
		r.Defaults[role] = defaultID(en, role)
	}
	r.EchoDecision = echoDecisionSummary(p)
	for flow := uintptr(0); flow < 2; flow++ {
		ds, e := listEndpoints(en, flow, 15)
		if e != nil {
			return e
		}
		for _, d := range ds {
			p := endpointReport{renderEndpoint: d, Sessions: []sessionReport{}}
			if d.State == 1 {
				var dev uintptr
				if hrOK(comCall(en, 5, uintptr(unsafe.Pointer(ptrUTF16(d.ID))), uintptr(unsafe.Pointer(&dev)))) && dev != 0 {
					var volume uintptr
					if hrOK(comCall(dev, 3, uintptr(unsafe.Pointer(&iidEndpointVolume)), 1, 0, uintptr(unsafe.Pointer(&volume)))) && volume != 0 {
						var muted int32
						comCall(volume, 15, uintptr(unsafe.Pointer(&muted)))
						p.EndpointMuted = muted != 0
						comCall(volume, 9, uintptr(unsafe.Pointer(&p.EndpointVolume)))
						release(volume)
					}
					var meter uintptr
					if hrOK(comCall(dev, 3, uintptr(unsafe.Pointer(&iidAudioMeter)), 1, 0, uintptr(unsafe.Pointer(&meter)))) && meter != 0 {
						comCall(meter, 3, uintptr(unsafe.Pointer(&p.Peak)))
						release(meter)
					}
					if e = eachSession(dev, func(_ uintptr, s sessionReport) error { p.Sessions = append(p.Sessions, s); return nil }); e != nil {
						p.Error = e.Error()
					}
					release(dev)
				}
			}
			r.Endpoints = append(r.Endpoints, p)
		}
	}
	return atomicJSON(path, r)
}

func echoDecisionSummary(p preferences) string {
	if !p.Echo {
		return "echo_off"
	}
	if p.EchoRoute == nil {
		return "echo_route_not_confirmed_no_audio_muted"
	}
	s, e := inspectEcho(p.EchoRoute)
	if e != nil {
		return "echo_waiting: " + e.Error()
	}
	_, reason := confirmedEchoDecision(p.EchoRoute, snapshotFacts(s))
	if reason == "confirmed_duplicate_render_only" && s.Render.Muted {
		return reason + "_muted"
	}
	return reason
}

// One-shot recovery only, never called by the monitor. Requires the exact
// endpoint, instance and PID from read-only diagnostics; never sets mute TRUE.
func recoverSession(endpoint, instance string, pid uint32) error {
	if isRunning() {
		return errors.New("stop SonarFix before recovery")
	}
	if endpoint == "" || instance == "" || pid == 0 {
		return errors.New("exact recovery identifiers required")
	}
	en, e := createEnumerator()
	if e != nil {
		return e
	}
	defer release(en)
	var d uintptr
	if !hrOK(comCall(en, 5, uintptr(unsafe.Pointer(ptrUTF16(endpoint))), uintptr(unsafe.Pointer(&d)))) || d == 0 {
		return errors.New("recovery endpoint missing")
	}
	defer release(d)
	f := endpointFlow(d)
	if (f != 0 && f != 1) || deviceID(d) != endpoint || !sonarMic(deviceName(d)) || !strings.Contains(strings.ToLower(deviceIdentity(d)), "steelseries") {
		return errors.New("recovery endpoint identity rejected")
	}
	found := false
	e = eachSession(d, func(vol uintptr, s sessionReport) error {
		if s.Instance != instance || s.PID != pid || s.Executable != "discord.exe" || s.Endpoint != endpoint {
			return nil
		}
		found = true
		if s.Muted {
			if !hrOK(comCall(vol, 5, 0, 0)) {
				return errors.New("session unmute failed")
			}
			var muted int32
			if !hrOK(comCall(vol, 6, uintptr(unsafe.Pointer(&muted)))) || muted != 0 {
				return errors.New("session unmute not verified")
			}
		}
		return nil
	})
	if e != nil {
		return e
	}
	if !found {
		return errors.New("session changed or disappeared; recovery skipped")
	}
	return nil
}

// Read-only observations for the selected echo feature. Sessions are identified
// and logged, but neither capture nor render identity proves a duplicate route.
func observeEchoSessions() (string, error) {
	en, e := createEnumerator()
	if e != nil {
		return "", e
	}
	defer release(en)
	var report strings.Builder
	for flow := uintptr(0); flow < 2; flow++ {
		ds, e := listEndpoints(en, flow, 1)
		if e != nil {
			return "", e
		}
		for _, ep := range ds {
			if !sonarMic(ep.Name) || !strings.Contains(strings.ToLower(ep.Identity), "steelseries") {
				continue
			}
			var d uintptr
			if !hrOK(comCall(en, 5, uintptr(unsafe.Pointer(ptrUTF16(ep.ID))), uintptr(unsafe.Pointer(&d)))) || d == 0 {
				continue
			}
			e = eachSession(d, func(_ uintptr, s sessionReport) error {
				if s.Executable == "discord.exe" {
					fmt.Fprintf(&report, "endpoint=%s flow=%d pid=%d instance=%q state=%d muted=%t decision=%s; ", s.Endpoint, s.Flow, s.PID, s.Instance, s.State, s.Muted, s.Reason)
				}
				return nil
			})
			release(d)
			if e != nil {
				return "", e
			}
		}
	}
	if report.Len() == 0 {
		return "no Discord session on a verified Sonar Microphone endpoint; no audio changed", nil
	}
	return report.String(), nil
}

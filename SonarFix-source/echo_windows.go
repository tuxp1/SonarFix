//go:build windows

package main

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type ownedRenderMute struct {
	Endpoint, Instance, ExecutablePath, HardwareIdentity string
	PID                                                  uint32
	Created                                              uint64
}
type echoSnapshot struct {
	Render, Capture                      sessionReport
	RenderIdentity, CaptureIdentity      string
	CaptureEndpointMuted                 bool
	CaptureEndpointVolume                float32
	IndependentPlayback, TargetIsDefault bool
}

func facts(s sessionReport) sessionFacts {
	return sessionFacts{s.Endpoint, s.Instance, s.Executable, s.Flow, s.State, s.PID}
}
func processCreated(pid uint32) uint64 {
	h, e := syscall.OpenProcess(processQueryLimitedInformation, false, pid)
	if e != nil {
		return 0
	}
	defer syscall.CloseHandle(h)
	var created, exit, kernel, user syscall.Filetime
	if e = syscall.GetProcessTimes(h, &created, &exit, &kernel, &user); e != nil {
		return 0
	}
	return uint64(created.HighDateTime)<<32 | uint64(created.LowDateTime)
}
func deviceByID(en uintptr, id string) (uintptr, error) {
	var d uintptr
	if !hrOK(comCall(en, 5, uintptr(unsafe.Pointer(ptrUTF16(id))), uintptr(unsafe.Pointer(&d)))) || d == 0 {
		return 0, errors.New("endpoint unavailable")
	}
	return d, nil
}
func endpointVolumeState(d uintptr) (bool, float32, error) {
	var v uintptr
	if !hrOK(comCall(d, 3, uintptr(unsafe.Pointer(&iidEndpointVolume)), 1, 0, uintptr(unsafe.Pointer(&v)))) || v == 0 {
		return false, 0, errors.New("capture guard unavailable")
	}
	defer release(v)
	var muted int32
	var volume float32
	if !hrOK(comCall(v, 15, uintptr(unsafe.Pointer(&muted)))) || !hrOK(comCall(v, 9, uintptr(unsafe.Pointer(&volume)))) {
		return false, 0, errors.New("capture guard unreadable")
	}
	return muted != 0, volume, nil
}

var errRouteChanged = errors.New("route endpoint changed or unavailable")

// Read-only snapshot of active endpoints and sessions for discoverEchoRoute.
func collectEndpointFacts() ([]endpointFacts, []string, error) {
	en, e := createEnumerator()
	if e != nil {
		return nil, nil, e
	}
	defer release(en)
	defaults := []string{}
	for role := uint32(0); role < 3; role++ {
		defaults = append(defaults, defaultID(en, role))
	}
	result := []endpointFacts{}
	for flow := uintptr(0); flow < 2; flow++ {
		ds, e := listEndpoints(en, flow, 1)
		if e != nil {
			return nil, nil, e
		}
		for _, d := range ds {
			dev, e := deviceByID(en, d.ID)
			if e != nil {
				continue
			}
			ep := endpointFacts{ID: d.ID, Name: d.Name, Identity: d.Identity, Flow: d.Flow}
			if flow == 1 {
				// An unreadable capture guard counts as muted, so it never qualifies.
				muted, _, e := endpointVolumeState(dev)
				ep.Muted = muted || e != nil
			}
			e = eachSession(dev, func(_ uintptr, s sessionReport) error {
				ep.Sessions = append(ep.Sessions, discoverySession{facts(s), s.ExecutablePath, s.Muted})
				return nil
			})
			release(dev)
			if e != nil {
				// A hidden Sonar Microphone session could be a second candidate.
				if sonarMic(d.Name) {
					return nil, nil, e
				}
				continue
			}
			result = append(result, ep)
		}
	}
	return result, defaults, nil
}

// adoptDiscoveredRoute stores a newly discovered route only after it passes
// the same inspection and confirmedEchoDecision the mute path uses.
func adoptDiscoveredRoute(f *fixer, p *preferences) (bool, string) {
	eps, defaults, e := collectEndpointFacts()
	if e != nil {
		return false, e.Error()
	}
	route, reason := discoverEchoRoute(eps, defaults, time.Now().Format(time.RFC3339))
	if route == nil {
		return false, reason
	}
	old := p.EchoRoute
	if old != nil && old.RenderID == route.RenderID && old.CaptureID == route.CaptureID && old.HardwareIdentity == route.HardwareIdentity && strings.EqualFold(old.ExecutablePath, route.ExecutablePath) {
		return false, "route_unchanged"
	}
	s, e := inspectEcho(route)
	if e != nil {
		return false, e.Error()
	}
	if ok, reason := confirmedEchoDecision(route, snapshotFacts(s)); !ok {
		return false, reason
	}
	p.EchoRoute = route
	if e = savePreferences(*p); e != nil {
		p.EchoRoute = old
		return false, e.Error()
	}
	f.logf("ECHO ROUTE AUTO-DETECTED render=%s capture=%s discord=%q", route.RenderID, route.CaptureID, route.ExecutablePath)
	return true, "auto_detected"
}
func inspectEcho(route *echoRoute) (echoSnapshot, error) {
	var result echoSnapshot
	if route == nil {
		return result, errors.New("route not confirmed")
	}
	en, e := createEnumerator()
	if e != nil {
		return result, e
	}
	defer release(en)
	r, e := deviceByID(en, route.RenderID)
	if e != nil {
		return result, errRouteChanged
	}
	defer release(r)
	c, e := deviceByID(en, route.CaptureID)
	if e != nil {
		return result, errRouteChanged
	}
	defer release(c)
	if endpointFlow(r) != 0 || endpointFlow(c) != 1 || !sonarMic(deviceName(r)) || !sonarMic(deviceName(c)) {
		return result, errRouteChanged
	}
	result.RenderIdentity = deviceIdentity(r)
	result.CaptureIdentity = deviceIdentity(c)
	renders := []sessionReport{}
	captures := []sessionReport{}
	if e = eachSession(r, func(_ uintptr, s sessionReport) error {
		if s.Executable == "discord.exe" && s.State == 1 {
			renders = append(renders, s)
		}
		return nil
	}); e != nil {
		return result, e
	}
	if e = eachSession(c, func(_ uintptr, s sessionReport) error {
		if s.Executable == "discord.exe" && s.State == 1 {
			captures = append(captures, s)
		}
		return nil
	}); e != nil {
		return result, e
	}
	if len(renders) != 1 || len(captures) != 1 {
		return result, errors.New("waiting for one unambiguous Discord broadcast and capture session")
	}
	result.Render = renders[0]
	result.Capture = captures[0]
	result.CaptureEndpointMuted, result.CaptureEndpointVolume, e = endpointVolumeState(c)
	if e != nil {
		return result, e
	}
	for role := uint32(0); role < 3; role++ {
		id := defaultID(en, role)
		if id == "" || id == route.RenderID {
			result.TargetIsDefault = true
		}
	}
	ds, e := listEndpoints(en, 0, 1)
	if e != nil {
		return result, e
	}
	for _, d := range ds {
		if d.ID == route.RenderID {
			continue
		}
		dev, e := deviceByID(en, d.ID)
		if e != nil {
			continue
		}
		e = eachSession(dev, func(_ uintptr, s sessionReport) error {
			if s.PID == result.Render.PID && s.State == 1 && s.Executable == "discord.exe" && !s.Muted && strings.EqualFold(s.ExecutablePath, route.ExecutablePath) {
				result.IndependentPlayback = true
			}
			return nil
		})
		release(dev)
		if e != nil {
			continue
		}
	}
	return result, nil
}
func snapshotFacts(s echoSnapshot) echoFacts {
	return echoFacts{facts(s.Render), facts(s.Capture), s.RenderIdentity, s.CaptureIdentity, s.Render.ExecutablePath, s.Capture.Muted || s.CaptureEndpointMuted, s.IndependentPlayback, s.TargetIsDefault}
}

// The only shipping SetMute(TRUE) call. It requires the stored manual A/B
// confirmation plus current endpoint, direction, process and capture guards.
func muteConfirmedRender(route *echoRoute, before echoSnapshot, vol uintptr) error {
	if ok, reason := confirmedEchoDecision(route, snapshotFacts(before)); !ok {
		return errors.New(reason)
	}
	if !hrOK(comCall(vol, 5, 1, 0)) {
		return errors.New("render session mute failed")
	}
	after, e := inspectEcho(route)
	ok, _ := confirmedEchoDecision(route, snapshotFacts(after))
	if e != nil || !ok || after.Render.Instance != before.Render.Instance || after.Capture.Instance != before.Capture.Instance || after.Capture.Muted || after.CaptureEndpointMuted || after.Capture.Volume != before.Capture.Volume || after.CaptureEndpointVolume != before.CaptureEndpointVolume {
		comCall(vol, 5, 0, 0)
		return errors.New("capture guard changed; render change rolled back")
	}
	var muted int32
	if !hrOK(comCall(vol, 6, uintptr(unsafe.Pointer(&muted)))) || muted == 0 {
		return errors.New("render mute not verified")
	}
	return nil
}
func restoreEchoMutes(f *fixer, p *preferences) error {
	if len(p.EchoMutes) == 0 {
		return nil
	}
	en, e := createEnumerator()
	if e != nil {
		return e
	}
	defer release(en)
	remaining := []ownedRenderMute{}
	for _, change := range p.EchoMutes {
		// A terminated process/session needs no restoration. Never reuse a PID.
		if change.Created == 0 || processCreated(change.PID) != change.Created || !strings.EqualFold(executablePath(change.PID), change.ExecutablePath) {
			continue
		}
		d, e := deviceByID(en, change.Endpoint)
		if e != nil {
			remaining = append(remaining, change)
			continue
		}
		if endpointFlow(d) != 0 || !strings.HasPrefix(change.Endpoint, "{0.0.0.") || deviceIdentity(d) != change.HardwareIdentity || !sonarMic(deviceName(d)) {
			release(d)
			remaining = append(remaining, change)
			continue
		}
		e = eachSession(d, func(vol uintptr, s sessionReport) error {
			if s.Instance != change.Instance || s.PID != change.PID || s.Executable != "discord.exe" {
				return nil
			}
			if s.Muted {
				if !hrOK(comCall(vol, 5, 0, 0)) {
					return errors.New("owned render unmute failed")
				}
				var muted int32
				if !hrOK(comCall(vol, 6, uintptr(unsafe.Pointer(&muted)))) || muted != 0 {
					return errors.New("owned render unmute not verified")
				}
			}
			f.logf("ECHO RESTORED render=%s instance=%q", s.Endpoint, s.Instance)
			return nil
		})
		release(d)
		if e != nil {
			remaining = append(remaining, change)
		}
	}
	p.EchoMutes = remaining
	if e = savePreferences(*p); e != nil {
		return e
	}
	if len(remaining) > 0 {
		return errors.New("echo restoration pending; journal retained")
	}
	return nil
}
func applyEchoMode(f *fixer, p *preferences) (string, error) {
	if !p.Echo {
		if e := restoreEchoMutes(f, p); e != nil {
			return "echo_off_restore_pending", e
		}
		return "echo_off", nil
	}
	if p.EchoRoute == nil {
		if ok, reason := adoptDiscoveredRoute(f, p); !ok {
			if e := restoreEchoMutes(f, p); e != nil {
				return "echo_detecting_restore_pending", e
			}
			return "echo_detecting: " + reason, nil
		}
	}
	s, e := inspectEcho(p.EchoRoute)
	// A route that validates is never replaced; only a stale one is rediscovered.
	stale := errors.Is(e, errRouteChanged)
	if e == nil {
		if ok, reason := confirmedEchoDecision(p.EchoRoute, snapshotFacts(s)); !ok && routeRediscoverable(reason) {
			stale = true
		}
	}
	if stale {
		if ok, _ := adoptDiscoveredRoute(f, p); ok {
			s, e = inspectEcho(p.EchoRoute)
		}
	}
	if e != nil {
		if len(p.EchoMutes) > 0 {
			if restore := restoreEchoMutes(f, p); restore != nil {
				return "echo_waiting_restore_pending", restore
			}
		}
		return "echo_waiting: " + e.Error(), nil
	}
	if ok, reason := confirmedEchoDecision(p.EchoRoute, snapshotFacts(s)); !ok {
		if e = restoreEchoMutes(f, p); e != nil {
			return reason, e
		}
		return reason, nil
	}
	// Restore obsolete owned instances, while retaining ownership of this one.
	current := []ownedRenderMute{}
	old := []ownedRenderMute{}
	for _, m := range p.EchoMutes {
		if m.Endpoint == s.Render.Endpoint && m.Instance == s.Render.Instance && m.PID == s.Render.PID && m.Created == processCreated(s.Render.PID) {
			current = append(current, m)
		} else {
			old = append(old, m)
		}
	}
	if len(old) > 0 {
		p.EchoMutes = old
		if e = restoreEchoMutes(f, p); e != nil {
			p.EchoMutes = append(p.EchoMutes, current...)
			if save := savePreferences(*p); save != nil {
				f.logf("Echo journal save failed: %v", save)
			}
			return "old_echo_restoration_pending", e
		}
		p.EchoMutes = current
		if e = savePreferences(*p); e != nil {
			return "echo_journal_failed", e
		}
	}
	if s.Render.Muted {
		return "echo_duplicate_render_muted_capture_preserved", nil
	}
	created := processCreated(s.Render.PID)
	if created == 0 {
		return "echo_process_identity_unavailable", nil
	}
	change := ownedRenderMute{s.Render.Endpoint, s.Render.Instance, s.Render.ExecutablePath, s.RenderIdentity, s.Render.PID, created}
	if len(current) == 0 {
		p.EchoMutes = append(p.EchoMutes, change)
		if e = savePreferences(*p); e != nil {
			return "echo_journal_failed", e
		}
	}
	en, e := createEnumerator()
	if e != nil {
		return "echo_enumerator_failed", e
	}
	defer release(en)
	d, e := deviceByID(en, s.Render.Endpoint)
	if e != nil {
		return "echo_render_disappeared", e
	}
	defer release(d)
	found := false
	e = eachSession(d, func(vol uintptr, now sessionReport) error {
		if now.Instance != s.Render.Instance || now.PID != s.Render.PID {
			return nil
		}
		found = true
		// Re-read all guards immediately before the write, then revalidate the
		// exact native session identity held by this callback.
		fresh, e := inspectEcho(p.EchoRoute)
		if e != nil {
			return e
		}
		if fresh.Render.Instance != now.Instance || processCreated(now.PID) != created {
			return errors.New("session changed before mute")
		}
		return muteConfirmedRender(p.EchoRoute, fresh, vol)
	})
	if e != nil || !found {
		if restore := restoreEchoMutes(f, p); restore != nil {
			f.logf("Echo restore after skipped change: %v", restore)
		}
		if e == nil {
			e = errors.New("render session disappeared")
		}
		return "echo_change_skipped", e
	}
	f.logf("ECHO MUTED verified duplicate render=%s instance=%q pid=%d; capture=%s preserved", s.Render.Endpoint, s.Render.Instance, s.Render.PID, s.Capture.Endpoint)
	return "echo_duplicate_render_muted_capture_preserved", nil
}
func confirmEchoRoute(path string) error {
	if isRunning() {
		return errors.New("stop SonarFix before route confirmation")
	}
	var request struct {
		RenderID, CaptureID                string
		UserConfirmedEchoGoneAndMicWorking bool
	}
	raw, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	if e = json.Unmarshal(raw, &request); e != nil {
		return e
	}
	if !request.UserConfirmedEchoGoneAndMicWorking {
		return errors.New("explicit user A/B confirmation required")
	}
	en, e := createEnumerator()
	if e != nil {
		return e
	}
	defer release(en)
	if request.RenderID == "" {
		ds, e := listEndpoints(en, 0, 1)
		if e != nil {
			return e
		}
		matches := []string{}
		for _, d := range ds {
			if !sonarMic(d.Name) {
				continue
			}
			dev, e := deviceByID(en, d.ID)
			if e != nil {
				continue
			}
			eachSession(dev, func(_ uintptr, s sessionReport) error {
				if s.Executable == "discord.exe" && s.State == 1 && s.Muted {
					matches = append(matches, d.ID)
				}
				return nil
			})
			release(dev)
		}
		if len(matches) != 1 {
			return errors.New("mute the duplicate Discord mixer route while broadcasting, then confirm")
		}
		request.RenderID = matches[0]
	}
	r, e := deviceByID(en, request.RenderID)
	if e != nil {
		return e
	}
	defer release(r)
	var candidate sessionReport
	count := 0
	if e = eachSession(r, func(_ uintptr, s sessionReport) error {
		if s.Executable == "discord.exe" && s.State == 1 && s.Muted {
			candidate = s
			count++
		}
		return nil
	}); e != nil {
		return e
	}
	if count != 1 {
		return errors.New("manually muted Discord route is not unambiguous")
	}
	if request.CaptureID == "" {
		ds, e := listEndpoints(en, 1, 1)
		if e != nil {
			return e
		}
		matches := []string{}
		for _, d := range ds {
			if !sonarMic(d.Name) || d.Identity != deviceIdentity(r) {
				continue
			}
			dev, e := deviceByID(en, d.ID)
			if e != nil {
				continue
			}
			eachSession(dev, func(_ uintptr, s sessionReport) error {
				if s.Executable == "discord.exe" && s.PID == candidate.PID && s.State == 1 && !s.Muted {
					matches = append(matches, d.ID)
				}
				return nil
			})
			release(dev)
		}
		if len(matches) != 1 {
			return errors.New("active, unmuted primary microphone session is not unambiguous")
		}
		request.CaptureID = matches[0]
	}
	route := &echoRoute{RenderID: request.RenderID, CaptureID: request.CaptureID, HardwareIdentity: deviceIdentity(r), ExecutablePath: candidate.ExecutablePath, ConfirmedAt: time.Now().Format(time.RFC3339)}
	s, e := inspectEcho(route)
	if e != nil {
		return e
	}
	if ok, reason := confirmedEchoDecision(route, snapshotFacts(s)); !ok {
		return errors.New(reason)
	}
	p, e := loadPreferences()
	if e != nil {
		return e
	}
	p.EchoRoute = route
	p.Echo = true
	return savePreferences(p)
}

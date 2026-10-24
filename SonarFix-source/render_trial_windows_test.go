//go:build windows && echo_trial

package main

// Explicit, bounded A/B experiment. Excluded from the shipping worker and all
// default tests. Never changes capture volume or endpoint mute. The request is
// produced from live diagnostics during a user-reproduced broadcast.

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

type renderTrialRequest struct {
	Render, RenderInstance, Capture, CaptureInstance, Report string
	PID                                                      uint32
	Seconds                                                  int
}

func TestExplicitRenderEchoTrial(t *testing.T) {
	path := os.Getenv("SONARFIX_RENDER_TRIAL_REQUEST")
	if path == "" {
		t.Skip("explicit observed-session request required")
	}
	var r renderTrialRequest
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(raw, &r); e != nil {
		t.Fatal(e)
	}
	if r.Render == "" || r.RenderInstance == "" || r.Capture == "" || r.CaptureInstance == "" || r.Report == "" || r.PID == 0 || r.Seconds < 1 || r.Seconds > 45 {
		t.Fatal("incomplete bounded trial request")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if e = initCOM(); e != nil {
		t.Fatal(e)
	}
	defer procCoUninitialize.Call()
	en, e := createEnumerator()
	if e != nil {
		t.Fatal(e)
	}
	defer release(en)
	get := func(id string) uintptr {
		var d uintptr
		if !hrOK(comCall(en, 5, uintptr(unsafe.Pointer(ptrUTF16(id))), uintptr(unsafe.Pointer(&d)))) || d == 0 {
			t.Fatal("endpoint unavailable")
		}
		return d
	}
	render := get(r.Render)
	defer release(render)
	capture := get(r.Capture)
	defer release(capture)
	if endpointFlow(render) != 0 || endpointFlow(capture) != 1 || deviceID(render) != r.Render || deviceID(capture) != r.Capture || !sonarMic(deviceName(render)) || !sonarMic(deviceName(capture)) || deviceIdentity(render) != deviceIdentity(capture) {
		t.Fatal("render/capture identity rejected")
	}
	for role := uint32(0); role < 3; role++ {
		if defaultID(en, role) == r.Render {
			t.Fatal("candidate is a normal Windows playback default")
		}
	}
	readCapture := func() (sessionReport, bool) {
		var found sessionReport
		ok := false
		e := eachSession(capture, func(_ uintptr, s sessionReport) error {
			if s.Instance == r.CaptureInstance && s.PID == r.PID && s.Executable == "discord.exe" {
				found = s
				ok = true
			}
			return nil
		})
		return found, ok && e == nil
	}
	initial, ok := readCapture()
	if !ok || initial.Muted || initial.State != 1 {
		t.Fatal("primary capture is not confirmed active and unmuted")
	}
	var endpointVol uintptr
	if !hrOK(comCall(capture, 3, uintptr(unsafe.Pointer(&iidEndpointVolume)), 1, 0, uintptr(unsafe.Pointer(&endpointVol)))) || endpointVol == 0 {
		t.Fatal("capture guard unavailable")
	}
	defer release(endpointVol)
	var initialMute int32
	var initialVolume float32
	if !hrOK(comCall(endpointVol, 15, uintptr(unsafe.Pointer(&initialMute)))) || initialMute != 0 || !hrOK(comCall(endpointVol, 9, uintptr(unsafe.Pointer(&initialVolume)))) {
		t.Fatal("capture endpoint guard failed")
	}
	journal := map[string]interface{}{"request": r, "phase": "prepared", "original_render_mute": false, "capture_before": initial, "capture_endpoint_volume": initialVolume, "time": time.Now().Format(time.RFC3339)}
	found := false
	e = eachSession(render, func(vol uintptr, s sessionReport) error {
		if s.Instance != r.RenderInstance || s.PID != r.PID || s.Executable != "discord.exe" {
			return nil
		}
		found = true
		if s.Flow != 0 || s.State != 1 || s.Muted || s.Endpoint != r.Render {
			return fmt.Errorf("candidate changed; no audio changed")
		}
		if e := atomicJSON(r.Report, journal); e != nil {
			return e
		}
		defer func() {
			h := comCall(vol, 5, 0, 0)
			var muted int32
			read := comCall(vol, 6, uintptr(unsafe.Pointer(&muted)))
			journal["phase"] = "restored"
			journal["restore_verified"] = hrOK(h) && hrOK(read) && muted == 0
			if after, ok := readCapture(); ok {
				journal["capture_after"] = after
			}
			atomicJSON(r.Report, journal)
			if !hrOK(h) || !hrOK(read) || muted != 0 {
				t.Error("render session restoration could not be verified")
			}
		}()
		if !hrOK(comCall(vol, 5, 1, 0)) {
			return fmt.Errorf("temporary render-only mute failed")
		}
		var actual int32
		if !hrOK(comCall(vol, 6, uintptr(unsafe.Pointer(&actual)))) || actual != 1 {
			return fmt.Errorf("temporary render-only mute not confirmed")
		}
		journal["phase"] = "trial_active"
		journal["started"] = time.Now().Format(time.RFC3339)
		if e := atomicJSON(r.Report, journal); e != nil {
			return e
		}
		t.Logf("RENDER-ONLY TRIAL ACTIVE FOR %d SECONDS; primary capture guard enabled", r.Seconds)
		until := time.Now().Add(time.Duration(r.Seconds) * time.Second)
		for time.Now().Before(until) {
			var muted int32
			var level float32
			if !hrOK(comCall(endpointVol, 15, uintptr(unsafe.Pointer(&muted)))) || !hrOK(comCall(endpointVol, 9, uintptr(unsafe.Pointer(&level)))) || muted != initialMute || level != initialVolume {
				return fmt.Errorf("capture endpoint guard changed; trial aborted")
			}
			current, ok := readCapture()
			if !ok || current.Muted || current.Volume != initial.Volume {
				return fmt.Errorf("primary capture guard changed; trial aborted")
			}
			time.Sleep(75 * time.Millisecond)
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	if !found {
		t.Fatal("exact candidate session disappeared; no audio changed")
	}
}

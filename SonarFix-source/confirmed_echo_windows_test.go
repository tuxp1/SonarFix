//go:build windows

package main

import (
	"encoding/json"
	"os"
	"runtime"
	"testing"
)

// A user-confirmed active broadcast is required. Preserve the user's manual
// mixer state while exercising the shipping apply/restore implementation.
func TestConfirmedEchoLiveApplyRestore(t *testing.T) {
	path := os.Getenv("SONARFIX_CONFIRMED_ECHO_TEST")
	if path == "" {
		t.Skip("requires explicit user-confirmed live echo route")
	}
	var route echoRoute
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(raw, &route); e != nil {
		t.Fatal(e)
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if e = initCOM(); e != nil {
		t.Fatal(e)
	}
	defer procCoUninitialize.Call()
	before, e := inspectEcho(&route)
	if e != nil {
		t.Fatal(e)
	}
	if ok, reason := confirmedEchoDecision(&route, snapshotFacts(before)); !ok {
		t.Fatal(reason)
	}
	if !before.Render.Muted {
		t.Fatal("the confirmed manual mixer mute must still be present")
	}
	t.Setenv("SONARFIX_DATA_DIR", t.TempDir())
	p := preferences{configuration: configuration{Echo: true, Language: "en"}, Version: 3, EchoRoute: &route}
	f := &fixer{}
	if _, e = applyEchoMode(f, &p); e != nil {
		t.Fatal(e)
	}
	if len(p.EchoMutes) != 0 {
		t.Fatal("claimed a pre-existing manual mute")
	}
	t.Log("Pre-existing manual render mute is not owned: PASS")
	en, e := createEnumerator()
	if e != nil {
		t.Fatal(e)
	}
	defer release(en)
	d, e := deviceByID(en, route.RenderID)
	if e != nil {
		t.Fatal(e)
	}
	defer release(d)
	found := false
	e = eachSession(d, func(vol uintptr, s sessionReport) error {
		if s.Instance != before.Render.Instance || s.PID != before.Render.PID {
			return nil
		}
		found = true
		defer func() {
			if !hrOK(comCall(vol, 5, 1, 0)) {
				t.Error("could not restore the user's manual render mute")
			}
		}()
		if !hrOK(comCall(vol, 5, 0, 0)) {
			t.Fatal("could not prepare owned-mute test")
		}
		status, e := applyEchoMode(f, &p)
		if e != nil {
			t.Fatal(e)
		}
		if status != "echo_duplicate_render_muted_capture_preserved" || len(p.EchoMutes) != 1 {
			t.Fatal(status, p.EchoMutes)
		}
		after, e := inspectEcho(&route)
		if e != nil || !after.Render.Muted || after.Capture.Muted || after.CaptureEndpointMuted || after.Capture.Volume != before.Capture.Volume || after.CaptureEndpointVolume != before.CaptureEndpointVolume {
			t.Fatal("capture changed during confirmed render mute", e)
		}
		t.Log("Automatic confirmed-render mute, primary capture unchanged: PASS")
		if e = restoreEchoMutes(f, &p); e != nil {
			t.Fatal(e)
		}
		after, e = inspectEcho(&route)
		if e != nil || after.Render.Muted || after.Capture.Muted {
			t.Fatal("owned-mute restoration failed", e)
		}
		t.Log("Owned render mute restores without touching capture: PASS")
		if _, e = applyEchoMode(f, &p); e != nil {
			t.Fatal(e)
		}
		p.Echo = false
		if _, e = applyEchoMode(f, &p); e != nil {
			t.Fatal(e)
		}
		after, e = inspectEcho(&route)
		if e != nil || after.Render.Muted || after.Capture.Muted || len(p.EchoMutes) != 0 {
			t.Fatal("feature-off restoration failed", e)
		}
		t.Log("Turning the feature off restores only its owned mute: PASS")
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	if !found {
		t.Fatal("confirmed render instance disappeared")
	}
	after, e := inspectEcho(&route)
	if e != nil || !after.Render.Muted || after.Capture.Muted {
		t.Fatal("manual baseline not preserved", e)
	}
	t.Log("User's original mixer mute restored after tests: PASS")
}

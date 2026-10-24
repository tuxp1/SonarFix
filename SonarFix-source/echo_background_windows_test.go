//go:build windows

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestConfirmedEchoBackgroundLoop(t *testing.T) {
	path := os.Getenv("SONARFIX_ECHO_BACKGROUND_TEST")
	worker := os.Getenv("SONARFIX_ECHO_BACKGROUND_WORKER")
	if path == "" || worker == "" {
		t.Skip("requires user-confirmed route and installed worker")
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
	p, e := loadPreferences()
	if e != nil || p.EchoRoute == nil || !p.Echo || p.EchoRoute.RenderID != route.RenderID || len(p.EchoMutes) != 0 || !isRunning() {
		t.Fatal("worker/profile/manual baseline not ready", e)
	}
	before, e := inspectEcho(&route)
	if e != nil || !before.Render.Muted {
		t.Fatal("confirmed manually muted route missing", e)
	}
	if ok, reason := confirmedEchoDecision(&route, snapshotFacts(before)); !ok {
		t.Fatal(reason)
	}
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
			signalStop()
			deadline := time.Now().Add(8 * time.Second)
			for isRunning() && time.Now().Before(deadline) {
				time.Sleep(50 * time.Millisecond)
			}
			if !hrOK(comCall(vol, 5, 1, 0)) {
				t.Error("manual baseline restoration failed")
			}
			cmd := exec.Command(worker, "--background")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if e := cmd.Start(); e != nil {
				t.Error(e)
			} else {
				cmd.Process.Release()
			}
		}()
		if !hrOK(comCall(vol, 5, 0, 0)) {
			t.Fatal("could not prepare background reapply test")
		}
		deadline := time.Now().Add(7 * time.Second)
		applied := false
		for time.Now().Before(deadline) {
			now, e := inspectEcho(&route)
			p, pe := loadPreferences()
			if e == nil && pe == nil && now.Render.Muted && len(p.EchoMutes) == 1 {
				applied = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !applied {
			t.Fatal("background worker did not reapply the confirmed render mute")
		}
		now, e := inspectEcho(&route)
		if e != nil || now.Capture.Muted || now.CaptureEndpointMuted || now.Capture.Volume != before.Capture.Volume || now.CaptureEndpointVolume != before.CaptureEndpointVolume {
			t.Fatal("capture guard changed", e)
		}
		t.Log("Installed background worker re-applies only the confirmed duplicate render mute: PASS")
		signalStop()
		deadline = time.Now().Add(8 * time.Second)
		for isRunning() && time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
		}
		if isRunning() {
			t.Fatal("background worker stop timed out")
		}
		now, e = inspectEcho(&route)
		p, pe := loadPreferences()
		if e != nil || pe != nil || now.Render.Muted || now.Capture.Muted || len(p.EchoMutes) != 0 {
			t.Fatal("stop did not restore its owned render mute", e, pe)
		}
		t.Log("Stopping the installed worker restores its own mute; capture remains unmuted: PASS")
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	if !found {
		t.Fatal("live render instance disappeared")
	}
	t.Log("Original manual mute restored and installed worker restarted: PASS")
}

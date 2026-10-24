//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// Explicit opt-in hardware test. Requires user consent to temporarily redirect
// defaults and disable Game/Chat. No GG/Discord restart or microphone mutation.
func TestLiveWindowsEndpointLifecycle(t *testing.T) {
	if os.Getenv("SONARFIX_LIVE_TEST") != "1" {
		t.Skip("requires explicit live audio test consent")
	}
	output := os.Getenv("SONARFIX_TEST_OUTPUT")
	worker := os.Getenv("SONARFIX_TEST_WORKER")
	if output == "" || worker == "" {
		t.Fatal("live test arguments missing")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if e := initCOM(); e != nil {
		t.Fatal(e)
	}
	defer procCoUninitialize.Call()
	if isRunning() {
		t.Fatal("stop other SonarFix workers before live test")
	}
	t.Setenv("SONARFIX_DATA_DIR", t.TempDir())
	en, e := createEnumerator()
	if e != nil {
		t.Fatal(e)
	}
	defer release(en)
	policy, e := createPolicyConfig()
	if e != nil {
		t.Fatal(e)
	}
	defer release(policy)
	ds, e := listEndpoints(en, 0, 15)
	if e != nil {
		t.Fatal(e)
	}
	capture, e := listEndpoints(en, 1, 15)
	if e != nil {
		t.Fatal(e)
	}
	initialDefaults := map[uint32]string{}
	for r := uint32(0); r < 3; r++ {
		initialDefaults[r] = defaultID(en, r)
	}
	defer func() {
		signalStop()
		for i := 0; i < 50 && isRunning(); i++ {
			time.Sleep(100 * time.Millisecond)
		}
		for _, d := range ds {
			if validTarget(d) && (d.State == 1 || d.State == 2) {
				if e := setEndpointVisibility(policy, d.ID, d.State == 1); e != nil {
					t.Errorf("cleanup endpoint: %v", e)
				}
			}
		}
		for r := uint32(0); r < 3; r++ {
			if id := initialDefaults[r]; id != "" {
				if e := setDefaultEndpoint(policy, id, r); e != nil {
					t.Errorf("cleanup default: %v", e)
				}
			}
		}
	}()
	chat := ""
	for _, d := range ds {
		if validTarget(d) && canonicalSonarOutput(d.Name) == "steelseries sonar - chat" && d.State == 1 {
			chat = d.ID
		}
	}
	if chat == "" {
		t.Skip("test needs an initially enabled Sonar Chat endpoint")
	}
	p := preferences{configuration: configuration{Echo: true, DisableBoth: true, OutputID: output, AllowRedirect: true, Language: "en"}, Version: 3}
	f := &fixer{}
	invalid := p
	invalid.OutputID = "{missing-output}"
	if e = disableUnwantedEndpoints(f, &invalid); e == nil {
		t.Fatal("missing output was accepted")
	}
	needsConsent := false
	for _, d := range ds {
		if validTarget(d) {
			for _, id := range initialDefaults {
				if id == d.ID {
					needsConsent = true
				}
			}
		}
	}
	if needsConsent {
		refused := p
		refused.AllowRedirect = false
		if e = disableUnwantedEndpoints(f, &refused); e == nil {
			t.Fatal("default redirect without consent accepted")
		}
	}
	for _, d := range ds {
		if endpointState(en, d.ID) != d.State {
			t.Fatal("refused operation changed an endpoint")
		}
	}
	for r := uint32(0); r < 3; r++ {
		if defaultID(en, r) != initialDefaults[r] {
			t.Fatal("refused operation changed a default")
		}
	}
	t.Log("Missing physical output and missing default-switch consent fail without changes: PASS")
	if e = disableUnwantedEndpoints(f, &p); e != nil {
		t.Fatal(e)
	}
	for _, d := range ds {
		if validTarget(d) && d.State == 1 && endpointState(en, d.ID) != 2 {
			t.Fatal("target not disabled", d.Name)
		}
	}
	if e = restoreUnwantedEndpoints(f, &p); e != nil {
		t.Fatal(e)
	}
	for r := uint32(0); r < 3; r++ {
		if defaultID(en, r) != initialDefaults[r] {
			t.Fatalf("default role %d was not restored", r)
		}
	}
	t.Log("Game/Chat disable and restoration, including all defaults: PASS")
	// Simulate a device disabled before SonarFix: it must never claim ownership.
	for r := uint32(0); r < 3; r++ {
		if e = setDefaultEndpoint(policy, output, r); e != nil {
			t.Fatal(e)
		}
	}
	if e = setEndpointVisibility(policy, chat, false); e != nil {
		t.Fatal(e)
	}
	if e = disableUnwantedEndpoints(f, &p); e != nil {
		t.Fatal(e)
	}
	if containsID(p.Changed, chat) {
		t.Fatal("claimed an already disabled endpoint")
	}
	if e = restoreUnwantedEndpoints(f, &p); e != nil {
		t.Fatal(e)
	}
	if endpointState(en, chat) != 2 {
		t.Fatal("re-enabled a previously disabled endpoint")
	}
	if e = setEndpointVisibility(policy, chat, true); e != nil {
		t.Fatal(e)
	}
	t.Log("Previously disabled device remains disabled: PASS")
	p.Echo = true
	p.DisableBoth = true
	if e = savePreferences(p); e != nil {
		t.Fatal(e)
	}
	start := func() *exec.Cmd {
		c := exec.Command(worker, "--background")
		c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if e := c.Start(); e != nil {
			t.Fatal(e)
		}
		return c
	}
	first := start()
	defer func() { signalStop(); first.Wait() }()
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) && (!isRunning() || endpointState(en, chat) != 2) {
		time.Sleep(100 * time.Millisecond)
	}
	if !isRunning() || endpointState(en, chat) != 2 {
		t.Fatal("worker did not apply desired state")
	}
	second := start()
	done := make(chan error, 1)
	go func() { done <- second.Wait() }()
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(3 * time.Second):
		second.Process.Kill()
		t.Fatal("duplicate worker was not rejected")
	}
	t.Log("Duplicate worker exits: PASS")
	// Simulate GG's endpoint reactivation without restarting the user's GG.
	if e = setEndpointVisibility(policy, chat, true); e != nil {
		t.Fatal(e)
	}
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && endpointState(en, chat) != 2 {
		time.Sleep(100 * time.Millisecond)
	}
	if endpointState(en, chat) != 2 {
		t.Fatal("worker did not reapply selected endpoint state")
	}
	t.Log("Simulated GG endpoint reactivation is re-disabled: PASS")
	signalStop()
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && isRunning() {
		time.Sleep(100 * time.Millisecond)
	}
	if isRunning() {
		t.Fatal("worker stop failed")
	}
	p, e = loadPreferences()
	if e != nil {
		t.Fatal(e)
	}
	if e = restoreUnwantedEndpoints(f, &p); e != nil {
		t.Fatal(e)
	}
	for _, d := range capture {
		if endpointState(en, d.ID) != d.State {
			t.Fatal("capture endpoint changed", d.Name)
		}
	}
	for _, d := range ds {
		if !validTarget(d) && endpointState(en, d.ID) != d.State {
			t.Fatal("non-target output changed", d.Name)
		}
	}
	if e = writeDiagnostics(filepath.Join(os.Getenv("SONARFIX_DATA_DIR"), "final.json")); e != nil {
		t.Fatal(e)
	}
	t.Log("All capture and non-target render endpoint states unchanged: PASS")
}

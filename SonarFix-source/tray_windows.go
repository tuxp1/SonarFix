//go:build windows

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// UI preferences are separate from audio ownership/configuration. A missing or
// corrupt file never starts a UI and never prevents audio monitoring.
func trayRequested(raw []byte) bool {
	var p struct {
		ShowTray bool `json:"show_tray"`
	}
	return json.Unmarshal(raw, &p) == nil && p.ShowTray
}
func startOptionalTray() {
	d, e := homeDir()
	if e != nil {
		return
	}
	raw, e := os.ReadFile(filepath.Join(d, "ui-settings.json"))
	if e != nil || !trayRequested(raw) {
		return
	}
	exe, e := os.Executable()
	if e != nil {
		return
	}
	c := exec.Command(filepath.Join(filepath.Dir(exe), "SonarFix.Settings.exe"), "--tray")
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if c.Start() == nil {
		go c.Wait()
	}
}

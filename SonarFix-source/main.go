//go:build windows

// SonarFix worker. The installer and settings UI are separate executables.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

type fixer struct{ log *os.File }

func (f *fixer) logf(format string, args ...interface{}) {
	if f.log != nil {
		if s, e := f.log.Stat(); e == nil && s.Size() > 256*1024 {
			f.log.Truncate(0)
		}
		fmt.Fprintf(f.log, "%s %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
		f.log.Sync()
	}
}
func homeDir() (string, error) {
	if d := os.Getenv("SONARFIX_DATA_DIR"); d != "" {
		return d, nil
	}
	if d := os.Getenv("LOCALAPPDATA"); d != "" {
		return filepath.Join(d, "SonarFix"), nil
	}
	return "", errors.New("LOCALAPPDATA unavailable")
}
func logOpen() *os.File {
	d, e := homeDir()
	if e != nil {
		return nil
	}
	os.MkdirAll(d, 0700)
	p := filepath.Join(d, "SonarFix.log")
	if s, e := os.Stat(p); e == nil && s.Size() > 256*1024 {
		os.Remove(p + ".1")
		os.Rename(p, p+".1")
	}
	f, _ := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	return f
}
func setRun(value string) error {
	var key uintptr
	var disposition uint32
	r, _, _ := procRegCreateKeyExW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(ptrUTF16(runKey))), 0, 0, 0, keySetValue, 0, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&disposition)))
	if r != 0 {
		return fmt.Errorf("startup key: %d", r)
	}
	defer procRegCloseKey.Call(key)
	d := syscall.StringToUTF16(value)
	r, _, _ = procRegSetValueExW.Call(key, uintptr(unsafe.Pointer(ptrUTF16(regValue))), 0, regSZ, uintptr(unsafe.Pointer(&d[0])), uintptr(len(d)*2))
	if r != 0 {
		return fmt.Errorf("startup value: %d", r)
	}
	return approveStartup()
}

// Task Manager's Startup apps toggle lives in StartupApproved\Run: the first
// byte is even when enabled (02) and odd when the user disabled it (03).
const startupApprovedKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`

func startupApprovedDisabled(data []byte) bool { return len(data) > 0 && data[0]&1 == 1 }
func startupDisabledByUser() bool {
	var key syscall.Handle
	if syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, ptrUTF16(startupApprovedKey), 0, syscall.KEY_QUERY_VALUE, &key) != nil {
		return false
	}
	defer syscall.RegCloseKey(key)
	buf := make([]byte, 64)
	n, typ := uint32(len(buf)), uint32(0)
	if syscall.RegQueryValueEx(key, ptrUTF16(regValue), nil, &typ, &buf[0], &n) != nil {
		return false
	}
	return startupApprovedDisabled(buf[:n])
}

// Called only when the user turns startup on, so Task Manager agrees with it.
func approveStartup() error {
	var key uintptr
	var disposition uint32
	r, _, _ := procRegCreateKeyExW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(ptrUTF16(startupApprovedKey))), 0, 0, 0, keySetValue, 0, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&disposition)))
	if r != 0 {
		return fmt.Errorf("startup approval key: %d", r)
	}
	defer procRegCloseKey.Call(key)
	enabled := make([]byte, 12)
	enabled[0] = 2
	r, _, _ = procRegSetValueExW.Call(key, uintptr(unsafe.Pointer(ptrUTF16(regValue))), 0, regBinary, uintptr(unsafe.Pointer(&enabled[0])), uintptr(len(enabled)))
	if r != 0 {
		return fmt.Errorf("startup approval value: %d", r)
	}
	return nil
}
func deleteRun() error {
	var key uintptr
	r, _, _ := procRegOpenKeyExW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(ptrUTF16(runKey))), 0, keySetValue, uintptr(unsafe.Pointer(&key)))
	if r == 2 {
		return nil
	}
	if r != 0 {
		return fmt.Errorf("startup key: %d", r)
	}
	defer procRegCloseKey.Call(key)
	r, _, _ = procRegDeleteValueW.Call(key, uintptr(unsafe.Pointer(ptrUTF16(regValue))))
	if r != 0 && r != 2 {
		return fmt.Errorf("startup removal: %d", r)
	}
	return nil
}
func signalStop() {
	h, _, _ := procOpenEventW.Call(eventModifyState, 0, uintptr(unsafe.Pointer(ptrUTF16(stopEventName))))
	if h != 0 {
		procSetEvent.Call(h)
		syscall.CloseHandle(syscall.Handle(h))
	}
}
func isRunning() bool {
	h, _, _ := procOpenEventW.Call(eventModifyState, 0, uintptr(unsafe.Pointer(ptrUTF16(stopEventName))))
	if h == 0 {
		return false
	}
	syscall.CloseHandle(syscall.Handle(h))
	return true
}
func stopWorker() error {
	signalStop()
	for i := 0; i < 80 && isRunning(); i++ {
		time.Sleep(100 * time.Millisecond)
	}
	if isRunning() {
		return errors.New("SonarFix did not stop; no settings changed")
	}
	return nil
}
func background() error {
	m, _, e := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(ptrUTF16(mutexName))))
	if m == 0 {
		return errors.New("worker mutex unavailable")
	}
	defer syscall.CloseHandle(syscall.Handle(m))
	if e == syscall.Errno(errorAlreadyExists) {
		return nil
	}
	s, _, _ := procCreateEventW.Call(0, 1, 0, uintptr(unsafe.Pointer(ptrUTF16(stopEventName))))
	if s == 0 {
		return errors.New("stop event unavailable")
	}
	defer syscall.CloseHandle(syscall.Handle(s))
	f := &fixer{log: logOpen()}
	if f.log != nil {
		defer f.log.Close()
	}
	startOptionalTray()
	f.logf("SonarFix %s started; only verified duplicate render routes may be muted", appVersion)
	lastStatus := ""
	for {
		p, e := loadPreferences()
		status := "No features selected"
		discord := false
		if p.Echo {
			discord, _ = discordProcessPresent()
		}
		if e != nil {
			status = e.Error()
		} else {
			if noMonitoringWork(p) {
				break
			}
			if canIdleEcho(p, discord) {
				status = "echo_waiting_for_discord"
			} else {
				status, e = applyEchoMode(f, &p)
			}
			if e != nil {
				status += "; " + e.Error()
			}
			if p.DisableBoth {
				if gameChatNeedsApply(p) {
					e = disableUnwantedEndpoints(f, &p)
				} else {
					e = nil
				}
				if e != nil {
					status += "; " + e.Error()
				} else {
					status += "; Game/Chat selection applied"
				}
			}
		}
		if status != lastStatus {
			f.logf("STATUS %s", status)
			lastStatus = status
			writeStatus(status, true)
		}
		r, e := syscall.WaitForSingleObject(syscall.Handle(s), monitoringDelay(p, discord))
		if e != nil {
			return e
		}
		if r == waitObject0 {
			break
		}
	}
	if p, e := loadPreferences(); e == nil {
		if e = restoreEchoMutes(f, &p); e != nil {
			f.logf("Echo restore on stop: %v", e)
			return e
		}
	}
	f.logf("SonarFix stopped")
	// Owned mutes are restored at this point; don't leave a running status behind.
	writeStatus("Stopped", false)
	return nil
}
func writeStatus(s string, running bool) {
	d, e := homeDir()
	if e == nil {
		p, _ := loadPreferences()
		atomicJSON(filepath.Join(d, "status.json"), map[string]interface{}{"status": s, "updated": time.Now().Format(time.RFC3339), "session_muting": running && p.Echo && p.EchoRoute != nil})
	}
}
func main() {
	backgroundFlag := flag.Bool("background", false, "Run selected features")
	diagnose := flag.String("diagnose-file", "", "Write read-only JSON diagnostics")
	setupInfo := flag.String("setup-info", "", "Write installer information INI (read-only)")
	healthInfo := flag.String("health-info", "", "Write read-only audio device health JSON")
	stop := flag.Bool("stop", false, "Stop SonarFix")
	configure := flag.String("configure", "", "Read settings request JSON")
	confirmRoute := flag.String("confirm-echo-route", "", "Store explicit manual mixer A/B confirmation")
	prepare := flag.Bool("prepare-uninstall", false, "Stop and restore changes before uninstall")
	recoverEndpoint := flag.String("recover-endpoint", "", "Explicit v2 recovery endpoint ID")
	recoverInstance := flag.String("recover-instance", "", "Explicit v2 session instance ID")
	recoverPID := flag.Uint("recover-pid", 0, "Verified affected v2 process ID")
	flag.Parse()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	err := initCOM()
	if err == nil {
		defer procCoUninitialize.Call()
		switch {
		case *backgroundFlag:
			err = background()
		case *diagnose != "":
			err = writeDiagnostics(*diagnose)
		case *setupInfo != "":
			err = writeSetupInfo(*setupInfo)
		case *healthInfo != "":
			err = writeHealthInfo(*healthInfo)
		case *stop:
			err = stopWorker()
		case *prepare:
			err = prepareUninstall()
		case *configure != "":
			err = configureFrom(*configure)
		case *confirmRoute != "":
			err = confirmEchoRoute(*confirmRoute)
		case *recoverEndpoint != "":
			err = recoverSession(*recoverEndpoint, *recoverInstance, uint32(*recoverPID))
		default:
			err = errors.New("Use SonarFix Settings or the SonarFix installer")
		}
	}
	if err != nil {
		f := &fixer{log: logOpen()}
		f.logf("ERROR %v", err)
		if f.log != nil {
			f.log.Close()
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func prepareUninstall() error {
	if e := stopWorker(); e != nil {
		return e
	}
	if e := deleteRun(); e != nil {
		return e
	}
	p, e := loadPreferences()
	if e != nil {
		return e
	}
	p.DisableBoth = false
	p.Echo = false
	p.Startup = false
	if e = savePreferences(p); e != nil {
		return e
	}
	f := &fixer{log: logOpen()}
	if f.log != nil {
		defer f.log.Close()
	}
	if e = restoreEchoMutes(f, &p); e != nil {
		return e
	}
	return restoreUnwantedEndpoints(f, &p)
}
func configureFrom(path string) error {
	if e := stopWorker(); e != nil {
		return e
	}
	p, e := loadPreferences()
	if e != nil {
		return e
	}
	raw, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	var r configuration
	if e = json.Unmarshal(raw, &r); e != nil {
		return e
	}
	if r.Language != "en" && r.Language != "ar" {
		return errors.New("invalid language")
	}
	if r.DisableBoth {
		if e = validateOutput(r.OutputID); e != nil {
			return e
		}
	}
	p.Echo = r.Echo
	p.DisableBoth = r.DisableBoth
	p.OutputID = r.OutputID
	p.AllowRedirect = r.AllowRedirect
	p.Startup = r.Startup
	p.Language = r.Language
	p.Version = 3
	// Remove v2's executable from startup before an attempted upgrade can fail.
	// A new startup entry is written only after the requested options succeed.
	if e = deleteRun(); e != nil {
		return e
	}
	if e = savePreferences(p); e != nil {
		return e
	}
	f := &fixer{log: logOpen()}
	if f.log != nil {
		defer f.log.Close()
	}
	if _, e = applyEchoMode(f, &p); e != nil {
		return e
	}
	if p.DisableBoth {
		e = disableUnwantedEndpoints(f, &p)
	} else {
		e = restoreUnwantedEndpoints(f, &p)
	}
	if e != nil {
		return e
	}
	if p.Startup {
		exe, e := os.Executable()
		if e != nil {
			return e
		}
		return setRun("\"" + exe + "\" --background")
	}
	return deleteRun()
}

//go:build windows

// SonarFix: targeted Discord session mute for the SteelSeries Sonar microphone endpoint.
// Native Windows Core Audio interfaces only; no driver, dependency or GG modification.
package main

import (
	"fmt"

	"path/filepath"

	"strings"
	"syscall"

	"unsafe"
)

type guid struct {
	d1     uint32
	d2, d3 uint16
	d4     [8]byte
}
type propertyKey struct {
	fmtid guid
	pid   uint32
}

var (
	clsidMMDeviceEnumerator        = guid{0xbcde0395, 0xe52f, 0x467c, [8]byte{0x8e, 0x3d, 0xc4, 0x57, 0x92, 0x91, 0x69, 0x2e}}
	iidMMDeviceEnumerator          = guid{0xa95664d2, 0x9614, 0x4f35, [8]byte{0xa7, 0x46, 0xde, 0x8d, 0xb6, 0x36, 0x17, 0xe6}}
	iidAudioSessionManager2        = guid{0x77aa99a0, 0x1bd6, 0x484f, [8]byte{0x8b, 0xc7, 0x2c, 0x65, 0x4c, 0x9a, 0x9b, 0x6f}}
	iidAudioSessionControl2        = guid{0xbfb7ff88, 0x7239, 0x4fc9, [8]byte{0x8f, 0xa2, 0x07, 0xc9, 0x50, 0xbe, 0x9c, 0x6d}}
	iidSimpleAudioVolume           = guid{0x87ce5498, 0x68d6, 0x44e5, [8]byte{0x92, 0x15, 0x6d, 0xa4, 0x7e, 0xf8, 0x83, 0xd8}}
	pkeyDeviceFriendlyName         = propertyKey{guid{0xa45c254e, 0xdf1c, 0x4efd, [8]byte{0x80, 0x20, 0x67, 0xd1, 0x46, 0xa8, 0x50, 0xe0}}, 14}
	ole32                          = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx             = ole32.NewProc("CoInitializeEx")
	procCoUninitialize             = ole32.NewProc("CoUninitialize")
	procCoCreateInstance           = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree              = ole32.NewProc("CoTaskMemFree")
	procPropVariantClear           = ole32.NewProc("PropVariantClear")
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW               = kernel32.NewProc("CreateMutexW")
	procCreateEventW               = kernel32.NewProc("CreateEventW")
	procOpenEventW                 = kernel32.NewProc("OpenEventW")
	procSetEvent                   = kernel32.NewProc("SetEvent")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	user32                         = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW                = user32.NewProc("MessageBoxW")
	advapi32                       = syscall.NewLazyDLL("advapi32.dll")
	procRegCreateKeyExW            = advapi32.NewProc("RegCreateKeyExW")
	procRegSetValueExW             = advapi32.NewProc("RegSetValueExW")
	procRegOpenKeyExW              = advapi32.NewProc("RegOpenKeyExW")
	procRegDeleteValueW            = advapi32.NewProc("RegDeleteValueW")
	procRegCloseKey                = advapi32.NewProc("RegCloseKey")
)

const (
	clsctxInprocServer             = 1
	deviceStateActive              = 1
	stgmRead                       = 0
	vtLPWSTR                       = 31
	processQueryLimitedInformation = 0x1000
	hkeyCurrentUser                = 0x80000001
	keySetValue                    = 0x0002
	regSZ                          = 1
	regBinary                      = 3
	runKey                         = `Software\Microsoft\Windows\CurrentVersion\Run`
	regValue                       = "SonarFix"
	mutexName                      = `Local\SonarFix_DiscordSonarMic_v1_Mutex`
	stopEventName                  = `Local\SonarFix_DiscordSonarMic_v1_Stop`
	eventModifyState               = 0x0002
	mbYesNo                        = 0x0004 | 0x0020 // question icon
	mbInfo                         = 0x0040
	mbError                        = 0x0010
	idYes                          = 6
	waitObject0                    = 0
	waitTimeout                    = 0x102
	errorAlreadyExists             = 183
)

func ptrUTF16(s string) *uint16 { return syscall.StringToUTF16Ptr(s) }
func utf16z(p *uint16) string {
	if p == nil {
		return ""
	}
	n := 0
	for *(*uint16)(unsafe.Add(unsafe.Pointer(p), uintptr(n)*2)) != 0 && n < 32768 {
		n++
	}
	return syscall.UTF16ToString(unsafe.Slice(p, n))
}
func hrOK(h uintptr) bool { return int32(h) >= 0 }

// COM arguments can contain pointers to Go out-parameters. Keep them on the
// heap across this wrapper's allocations and the native call, including tests
// and future compiler versions with different stack-growth decisions.
//
//go:uintptrescapes
func comCall(p uintptr, slot uintptr, args ...uintptr) uintptr {
	if p == 0 {
		return 0x80004003
	}
	vtable := *(*uintptr)(unsafe.Pointer(p))
	addr := *(*uintptr)(unsafe.Pointer(vtable + slot*unsafe.Sizeof(uintptr(0))))
	all := make([]uintptr, 0, 1+len(args))
	all = append(all, p)
	all = append(all, args...)
	r, _, _ := syscall.SyscallN(addr, all...)
	return r
}
func release(p uintptr) {
	if p != 0 {
		comCall(p, 2)
	}
}
func queryInterface(p uintptr, iid *guid) uintptr {
	var q uintptr
	if hrOK(comCall(p, 0, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&q)))) {
		return q
	}
	return 0
}
func initCOM() error {
	r, _, _ := procCoInitializeEx.Call(0, 0) // COINIT_MULTITHREADED
	if !hrOK(r) {
		return fmt.Errorf("COM init failed (0x%08x)", uint32(r))
	}
	return nil
}
func createEnumerator() (uintptr, error) {
	var p uintptr
	r, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidMMDeviceEnumerator)), 0, clsctxInprocServer, uintptr(unsafe.Pointer(&iidMMDeviceEnumerator)), uintptr(unsafe.Pointer(&p)))
	if !hrOK(r) || p == 0 {
		return 0, fmt.Errorf("audio enumerator failed (0x%08x)", uint32(r))
	}
	return p, nil
}
func deviceName(p uintptr) string {
	var store uintptr
	if !hrOK(comCall(p, 4, stgmRead, uintptr(unsafe.Pointer(&store)))) || store == 0 {
		return ""
	}
	defer release(store)
	var pv [3]uintptr // PROPVARIANT is 24 bytes on Win64.
	if !hrOK(comCall(store, 5, uintptr(unsafe.Pointer(&pkeyDeviceFriendlyName)), uintptr(unsafe.Pointer(&pv[0])))) {
		return ""
	}
	defer procPropVariantClear.Call(uintptr(unsafe.Pointer(&pv[0])))
	if *(*uint16)(unsafe.Pointer(&pv[0])) != vtLPWSTR {
		return ""
	}
	return utf16z((*uint16)(unsafe.Pointer(pv[1])))
}
func deviceID(p uintptr) string {
	var id *uint16
	if !hrOK(comCall(p, 5, uintptr(unsafe.Pointer(&id)))) || id == nil {
		return ""
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(id)))
	return utf16z(id)
}
func getSessionInstance(c2 uintptr) string {
	var id *uint16
	if !hrOK(comCall(c2, 13, uintptr(unsafe.Pointer(&id)))) || id == nil {
		return ""
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(id)))
	return utf16z(id)
}
func executableName(pid uint32) string {
	return strings.ToLower(filepath.Base(executablePath(pid)))
}
func executablePath(pid uint32) string {
	if pid == 0 {
		return ""
	}
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, pid)
	if err != nil {
		return ""
	}
	defer syscall.CloseHandle(h)
	var buf [32768]uint16
	length := uint32(len(buf))
	r, _, _ := procQueryFullProcessImageNameW.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&length)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:length])
}

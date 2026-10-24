//go:build windows

// Reuses v2's endpoint policy COM interop, with checked direction, hardware
// identity, explicit output consent, transactional journals and verification.
// IPolicyConfig is undocumented: failures are reported and changes journalled.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

var (
	clsidPolicyConfig = guid{0x870af99c, 0x171d, 0x4f9e, [8]byte{0xaf, 0x0d, 0xe6, 0x3d, 0xf4, 0x0c, 0x2b, 0xc9}}
	iidPolicyConfig   = guid{0xf8679f50, 0x850a, 0x41cf, [8]byte{0x9c, 0x72, 0x43, 0x0f, 0x29, 0x02, 0x90, 0xc8}}
)

const deviceStateDisabled = 2

type configuration struct {
	Echo          bool   `json:"fix_discord_echo"`
	DisableBoth   bool   `json:"disable_game_and_chat"`
	OutputID      string `json:"physical_output_id"`
	AllowRedirect bool   `json:"allow_default_redirect"`
	Startup       bool   `json:"startup"`
	Language      string `json:"language"`
}
type defaultChange struct{ Original, Replacement string }
type preferences struct {
	configuration
	Version   int                      `json:"version"`
	Changed   []string                 `json:"devices_disabled_by_sonarfix"`
	Pending   []string                 `json:"pending_endpoint_changes,omitempty"`
	Defaults  map[uint32]defaultChange `json:"default_changes,omitempty"`
	EchoRoute *echoRoute               `json:"confirmed_echo_route,omitempty"`
	EchoMutes []ownedRenderMute        `json:"owned_render_mutes,omitempty"`
}

func settingsPath() (string, error) { d, e := homeDir(); return filepath.Join(d, "settings.json"), e }
func loadPreferences() (preferences, error) {
	p := preferences{configuration: configuration{Language: "en"}}
	name, e := settingsPath()
	if e != nil {
		return p, e
	}
	raw, e := os.ReadFile(name)
	if os.IsNotExist(e) {
		return p, nil
	}
	if e != nil {
		return p, e
	}
	if e = json.Unmarshal(raw, &p); e != nil {
		return p, fmt.Errorf("invalid settings: %w", e)
	}
	return p, nil
}
func atomicJSON(name string, value interface{}) error {
	if e := os.MkdirAll(filepath.Dir(name), 0700); e != nil {
		return e
	}
	raw, e := json.MarshalIndent(value, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(name), ".sonarfix-*.tmp")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, e = f.Write(append(raw, '\n')); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	// A concurrent reader (Settings refresh, diagnostics) can briefly hold the
	// target open without delete sharing, which makes Windows refuse a replace.
	for i := 0; ; i++ {
		if e = os.Rename(tmp, name); e == nil || i == 9 {
			return e
		}
		time.Sleep(50 * time.Millisecond)
	}
}
func savePreferences(p preferences) error {
	name, e := settingsPath()
	if e != nil {
		return e
	}
	return atomicJSON(name, p)
}
func containsID(list []string, id string) bool {
	for _, v := range list {
		if strings.EqualFold(v, id) {
			return true
		}
	}
	return false
}
func removeID(list []string, id string) []string {
	r := []string{}
	for _, v := range list {
		if !strings.EqualFold(v, id) {
			r = append(r, v)
		}
	}
	return r
}
func sonarUnwantedOutput(name string) bool { return canonicalSonarOutput(name) != "" }
func createPolicyConfig() (uintptr, error) {
	var p uintptr
	h, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidPolicyConfig)), 0, 1, uintptr(unsafe.Pointer(&iidPolicyConfig)), uintptr(unsafe.Pointer(&p)))
	if !hrOK(h) || p == 0 {
		return 0, fmt.Errorf("endpoint policy unavailable: %08x", uint32(h))
	}
	return p, nil
}
func setEndpointVisibility(policy uintptr, id string, enabled bool) error {
	var v uintptr
	if enabled {
		v = 1
	}
	// Modern IPolicyConfig has ResetDeviceFormat at slot 5. The v2 Vista
	// visibility call returned success without disabling on this Windows build.
	h := comCall(policy, 14, uintptr(unsafe.Pointer(ptrUTF16(id))), v)
	if !hrOK(h) {
		return fmt.Errorf("endpoint visibility failed: %08x", uint32(h))
	}
	return nil
}
func setDefaultEndpoint(policy uintptr, id string, role uint32) error {
	h := comCall(policy, 13, uintptr(unsafe.Pointer(ptrUTF16(id))), uintptr(role))
	if !hrOK(h) {
		return fmt.Errorf("default role %d: %08x", role, uint32(h))
	}
	return nil
}

type renderEndpoint struct {
	ID, Name, Identity string
	State              uint32
	Flow               int32
}

func validTarget(d renderEndpoint) bool {
	return d.Flow == 0 && strings.HasPrefix(d.ID, "{0.0.0.") && sonarUnwantedOutput(d.Name) && strings.Contains(strings.ToLower(d.Identity), "steelseries")
}
func physicalOutput(d renderEndpoint) bool {
	i := strings.ToLower(d.Identity)
	return d.Flow == 0 && d.State == 1 && strings.HasPrefix(d.ID, "{0.0.0.") && !strings.Contains(strings.ToLower(d.Name), "sonar") && (strings.Contains(i, `usb\vid_`) || strings.Contains(i, `hdaudio\`) || strings.Contains(i, `bthenum\`) || strings.Contains(i, `bthhfenum\`))
}
func listEndpoints(en uintptr, flow, mask uintptr) ([]renderEndpoint, error) {
	var coll uintptr
	h := comCall(en, 3, flow, mask, uintptr(unsafe.Pointer(&coll)))
	if !hrOK(h) || coll == 0 {
		return nil, fmt.Errorf("endpoint enumeration: %08x", uint32(h))
	}
	defer release(coll)
	var n uint32
	if !hrOK(comCall(coll, 3, uintptr(unsafe.Pointer(&n)))) {
		return nil, errors.New("endpoint count failed")
	}
	ds := []renderEndpoint{}
	for i := uint32(0); i < n; i++ {
		var d uintptr
		if !hrOK(comCall(coll, 4, uintptr(i), uintptr(unsafe.Pointer(&d)))) || d == 0 {
			continue
		}
		var state uint32
		actual := endpointFlow(d)
		if hrOK(comCall(d, 6, uintptr(unsafe.Pointer(&state)))) && actual == int32(flow) {
			ds = append(ds, renderEndpoint{deviceID(d), deviceName(d), deviceIdentity(d), state, actual})
		}
		release(d)
	}
	return ds, nil
}
func defaultID(en uintptr, role uint32) string {
	var d uintptr
	if !hrOK(comCall(en, 4, 0, uintptr(role), uintptr(unsafe.Pointer(&d)))) || d == 0 {
		return ""
	}
	defer release(d)
	return deviceID(d)
}
func validateOutput(id string) error {
	en, e := createEnumerator()
	if e != nil {
		return e
	}
	defer release(en)
	ds, e := listEndpoints(en, 0, 1)
	if e != nil {
		return e
	}
	for _, d := range ds {
		if d.ID == id && physicalOutput(d) {
			return nil
		}
	}
	return errors.New("selected physical output is not active; Game/Chat unchanged")
}
func endpointState(en uintptr, id string) uint32 {
	var d uintptr
	if !hrOK(comCall(en, 5, uintptr(unsafe.Pointer(ptrUTF16(id))), uintptr(unsafe.Pointer(&d)))) || d == 0 {
		return 0
	}
	defer release(d)
	var s uint32
	comCall(d, 6, uintptr(unsafe.Pointer(&s)))
	return s
}
func disableUnwantedEndpoints(f *fixer, p *preferences) error {
	if e := validateOutput(p.OutputID); e != nil {
		return e
	}
	en, e := createEnumerator()
	if e != nil {
		return e
	}
	defer release(en)
	ds, e := listEndpoints(en, 0, 3)
	if e != nil {
		return e
	}
	targets := map[string]bool{}
	for _, d := range ds {
		if validTarget(d) {
			targets[d.ID] = true
		}
	}
	if len(targets) == 0 {
		return errors.New("verified Sonar Game/Chat endpoints not found")
	}
	policy, e := createPolicyConfig()
	if e != nil {
		return e
	}
	defer release(policy)
	// Snapshot every role before changing any: Windows can couple console and
	// multimedia roles, so reading role 1 after setting role 0 loses its original.
	originalDefaults := map[uint32]string{}
	for role := uint32(0); role < 3; role++ {
		id := defaultID(en, role)
		if id == "" {
			return errors.New("cannot verify Windows output defaults")
		}
		originalDefaults[role] = id
		if targets[id] && !p.AllowRedirect {
			return errors.New("Game/Chat is a default output; consent to switch is required")
		}
	}
	if p.Defaults == nil {
		p.Defaults = map[uint32]defaultChange{}
	}
	for role, id := range originalDefaults {
		if targets[id] {
			if _, ok := p.Defaults[role]; !ok {
				p.Defaults[role] = defaultChange{id, p.OutputID}
			}
		}
	}
	if e = savePreferences(*p); e != nil {
		return e
	}
	for role := uint32(0); role < 3; role++ {
		id := originalDefaults[role]
		if !targets[id] {
			continue
		}
		if e = setDefaultEndpoint(policy, p.OutputID, role); e != nil {
			return e
		}
		if defaultID(en, role) != p.OutputID {
			return errors.New("default redirection could not be verified")
		}
		f.logf("DEFAULT role=%d from=%s to=%s", role, id, p.OutputID)
	}
	for _, d := range ds {
		if !validTarget(d) || d.State != 1 {
			continue
		}
		if e = validateOutput(p.OutputID); e != nil {
			return e
		}
		for role := uint32(0); role < 3; role++ {
			id := defaultID(en, role)
			if id == "" || targets[id] {
				return errors.New("output defaults changed; refusing to disable")
			}
		}
		if !containsID(p.Pending, d.ID) {
			p.Pending = append(p.Pending, d.ID)
		}
		if e = savePreferences(*p); e != nil {
			return e
		}
		if e = setEndpointVisibility(policy, d.ID, false); e != nil {
			return e
		}
		if endpointState(en, d.ID) != 2 {
			return errors.New("Windows did not confirm endpoint Disabled state")
		}
		if !containsID(p.Changed, d.ID) {
			p.Changed = append(p.Changed, d.ID)
		}
		p.Pending = removeID(p.Pending, d.ID)
		if e = savePreferences(*p); e != nil {
			return e
		}
		f.logf("DISABLED id=%s name=%q", d.ID, d.Name)
	}
	return nil
}
func restoreUnwantedEndpoints(f *fixer, p *preferences) error {
	if len(p.Changed) == 0 && len(p.Defaults) == 0 && len(p.Pending) == 0 {
		return nil
	}
	en, e := createEnumerator()
	if e != nil {
		return e
	}
	defer release(en)
	ds, e := listEndpoints(en, 0, 15)
	if e != nil {
		return e
	}
	policy, e := createPolicyConfig()
	if e != nil {
		return e
	}
	defer release(policy)
	for _, id := range append([]string(nil), p.Pending...) {
		if endpointState(en, id) == 1 || containsID(p.Changed, id) {
			p.Pending = removeID(p.Pending, id)
		}
	}
	remaining := []string{}
	for _, id := range p.Changed {
		found := false
		for _, d := range ds {
			if d.ID != id || !validTarget(d) {
				continue
			}
			found = true
			if d.State == 2 {
				if e = setEndpointVisibility(policy, id, true); e != nil || endpointState(en, id) != 1 {
					remaining = append(remaining, id)
					break
				}
				f.logf("RESTORED id=%s", id)
			}
			if d.State != 1 && d.State != 2 {
				remaining = append(remaining, id)
			}
			break
		}
		if !found {
			remaining = append(remaining, id)
		}
	}
	p.Changed = remaining
	// Snapshot current choices too: restoring Console can affect Multimedia.
	// Reapply unaffected/current user choices if Windows couples the roles.
	currentDefaults := map[uint32]string{}
	desiredDefaults := map[uint32]string{}
	for role := uint32(0); role < 3; role++ {
		currentDefaults[role] = defaultID(en, role)
		desiredDefaults[role] = currentDefaults[role]
	}
	for role := uint32(0); role < 3; role++ {
		c, exists := p.Defaults[role]
		if !exists {
			continue
		}
		if currentDefaults[role] != c.Replacement {
			delete(p.Defaults, role)
			continue
		}
		if endpointState(en, c.Original) != 1 {
			continue
		}
		desiredDefaults[role] = c.Original
	}
	for role := uint32(0); role < 3; role++ {
		if id := desiredDefaults[role]; id != "" && defaultID(en, role) != id {
			if e = setDefaultEndpoint(policy, id, role); e != nil {
				f.logf("Default restoration: %v", e)
			}
		}
	}
	for role, c := range p.Defaults {
		if desiredDefaults[role] == c.Original && defaultID(en, role) == c.Original {
			delete(p.Defaults, role)
		}
	}
	if e = savePreferences(*p); e != nil {
		return e
	}
	if len(p.Changed) > 0 || len(p.Defaults) > 0 || len(p.Pending) > 0 {
		return errors.New("some restoration is pending; journal retained in settings.json")
	}
	return nil
}

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestSourceRestrictsMuteToConfirmedRenderGuard(t *testing.T) {
	files, e := filepath.Glob("*.go")
	if e != nil {
		t.Fatal(e)
	}
	for _, file := range files {
		if filepath.Ext(file) != ".go" || len(file) >= 8 && file[len(file)-8:] == "_test.go" {
			continue
		}
		f, e := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if e != nil {
			t.Fatal(e)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				c, ok := n.(*ast.CallExpr)
				if !ok || len(c.Args) < 3 {
					return true
				}
				name, ok := c.Fun.(*ast.Ident)
				if !ok || name.Name != "comCall" {
					return true
				}
				slot, sok := c.Args[1].(*ast.BasicLit)
				value, vok := c.Args[2].(*ast.BasicLit)
				if sok && vok && slot.Value == "5" && value.Value == "1" && (file != "echo_windows.go" || fn.Name.Name != "muteConfirmedRender") {
					t.Errorf("SetMute(TRUE) escaped the confirmed-render guard in %s", file)
				}
				return true
			})
		}
	}
}

func TestOnlyConfirmedDuplicateRenderCanMute(t *testing.T) {
	route := &echoRoute{RenderID: "{0.0.0.00000000}.{render}", CaptureID: "{0.0.1.00000000}.{capture}", HardwareIdentity: "ROOT\\MEDIA | SteelSeries Sonar Virtual Audio Device", ExecutablePath: `C:\Discord\Discord.exe`, ConfirmedAt: "user-confirmed"}
	base := echoFacts{Render: sessionFacts{route.RenderID, "render-instance", "discord.exe", 0, 1, 42}, Capture: sessionFacts{route.CaptureID, "capture-instance", "discord.exe", 1, 1, 42}, RenderIdentity: route.HardwareIdentity, CaptureIdentity: route.HardwareIdentity, ExecutablePath: route.ExecutablePath, IndependentPlayback: true}
	if ok, reason := confirmedEchoDecision(route, base); !ok {
		t.Fatal(reason)
	}
	if ok, _ := confirmedEchoDecision(nil, base); ok {
		t.Fatal("unconfirmed route muted")
	}
	for name, alter := range map[string]func(*echoFacts){
		"capture direction":  func(f *echoFacts) { f.Render.Flow = 1 },
		"unknown direction":  func(f *echoFacts) { f.Render.Flow = -1 },
		"capture replaced":   func(f *echoFacts) { f.Capture.Endpoint = "different" },
		"render replaced":    func(f *echoFacts) { f.Render.Endpoint = "different" },
		"hardware changed":   func(f *echoFacts) { f.RenderIdentity = "other driver" },
		"binary changed":     func(f *echoFacts) { f.ExecutablePath = `C:\Other\Discord.exe` },
		"capture muted":      func(f *echoFacts) { f.CaptureMuted = true },
		"capture inactive":   func(f *echoFacts) { f.Capture.State = 0 },
		"PID mismatch":       func(f *echoFacts) { f.Capture.PID = 99 },
		"missing instance":   func(f *echoFacts) { f.Render.Instance = "" },
		"normal output":      func(f *echoFacts) { f.TargetIsDefault = true },
		"no separate output": func(f *echoFacts) { f.IndependentPlayback = false },
	} {
		t.Run(name, func(t *testing.T) {
			changed := base
			alter(&changed)
			if ok, _ := confirmedEchoDecision(route, changed); ok {
				t.Fatal("unsafe mute accepted")
			}
		})
	}
}

func TestNeverMuteCaptureOrAmbiguousDiscord(t *testing.T) {
	for _, flow := range []int32{-1, 0, 1, 2} {
		for _, state := range []int32{-1, 0, 1, 2} {
			s := sessionFacts{"endpoint", "instance", "discord.exe", flow, state, 9360}
			ok, reason := muteDecision(s)
			if ok || reason == "" {
				t.Fatalf("unsafe policy: %+v", s)
			}
		}
	}
}
func TestMissingIdentityNeverAllowsMute(t *testing.T) {
	for _, s := range []sessionFacts{{}, {Flow: 0, State: 1, PID: 9360, Executable: "discord.exe"}, {Endpoint: "id", Instance: "session", Flow: 0, State: 1}} {
		if ok, _ := muteDecision(s); ok {
			t.Fatal(s)
		}
	}
}
func TestOnlyGamingAndChat(t *testing.T) {
	for _, n := range []string{"SteelSeries Sonar - Gaming", "SteelSeries Sonar - Chat (SteelSeries Sonar Virtual Audio Device)", "SteelSeries Sonar - Game"} {
		if !sonarUnwantedOutput(n) {
			t.Fatal(n)
		}
	}
	for _, n := range []string{"SteelSeries Sonar - Microphone", "SteelSeries Sonar - Media", "SteelSeries Sonar - Aux", "SteelSeries Sonar - Stream", "Headphones (Arctis Nova Pro Wireless)", "Fake SteelSeries Sonar - Chat", "SteelSeries Sonar - Chat Evil"} {
		if sonarUnwantedOutput(n) {
			t.Fatal(n)
		}
	}
}
func TestEndpointIdentityAndDirection(t *testing.T) {
	d := renderEndpoint{ID: "{0.0.0.00000000}.{test}", Name: "SteelSeries Sonar - Chat", Identity: "ROOT\\MEDIA | SteelSeries Sonar Virtual Audio Device", Flow: 0, State: 1}
	if !validTarget(d) {
		t.Fatal("valid Sonar render endpoint rejected")
	}
	d.Flow = 1
	if validTarget(d) {
		t.Fatal("capture selected")
	}
	d.Flow = 0
	d.ID = "{0.0.1.00000000}.{test}"
	if validTarget(d) {
		t.Fatal("capture ID selected")
	}
	d.ID = "{0.0.0.00000000}.{test}"
	d.Identity = "USB headset"
	if validTarget(d) {
		t.Fatal("wrong hardware selected")
	}
}
func TestOutputMustBePhysicalAndActive(t *testing.T) {
	d := renderEndpoint{ID: "{0.0.0.00000000}.{test}", Name: "Headphones", Identity: `{1}.USB\VID_1038`, State: 1, Flow: 0}
	if !physicalOutput(d) {
		t.Fatal("physical USB rejected")
	}
	d.State = 2
	if physicalOutput(d) {
		t.Fatal("disabled output selected")
	}
	d.State = 1
	d.Flow = 1
	if physicalOutput(d) {
		t.Fatal("microphone selected")
	}
	d.Flow = 0
	d.Identity = `ROOT\MEDIA`
	if physicalOutput(d) {
		t.Fatal("virtual device selected")
	}
}
func TestPreferencesRoundTripAndLegacyMigration(t *testing.T) {
	d := t.TempDir()
	t.Setenv("SONARFIX_DATA_DIR", d)
	if e := os.WriteFile(filepath.Join(d, "settings.json"), []byte(`{"disable_game_and_chat":true,"devices_disabled_by_sonarfix":["old-id"]}`), 0600); e != nil {
		t.Fatal(e)
	}
	p, e := loadPreferences()
	if e != nil || !p.DisableBoth || !containsID(p.Changed, "old-id") || p.Echo {
		t.Fatalf("legacy state lost: %+v %v", p, e)
	}
	for _, v := range [][2]bool{{true, false}, {false, true}, {true, true}, {false, false}} {
		p.Echo = v[0]
		p.DisableBoth = v[1]
		p.Version = 3
		if e = savePreferences(p); e != nil {
			t.Fatal(e)
		}
		r, e := loadPreferences()
		if e != nil || r.Echo != v[0] || r.DisableBoth != v[1] || !containsID(r.Changed, "old-id") {
			t.Fatal(r, e)
		}
	}
}
func TestCorruptSettingsFailWithoutReset(t *testing.T) {
	d := t.TempDir()
	t.Setenv("SONARFIX_DATA_DIR", d)
	f := filepath.Join(d, "settings.json")
	os.WriteFile(f, []byte("{broken"), 0600)
	if _, e := loadPreferences(); e == nil {
		t.Fatal("corruption ignored")
	}
	raw, _ := os.ReadFile(f)
	if string(raw) != "{broken" {
		t.Fatal("corrupt file overwritten")
	}
}
func TestOwnershipIDs(t *testing.T) {
	a := []string{"own", "already-disabled"}
	a = removeID(a, "own")
	if containsID(a, "own") || !containsID(a, "already-disabled") {
		t.Fatal(a)
	}
}

func discoveryFixture() ([]endpointFacts, []string) {
	identity := `{1}.ROOT\MEDIA\0000 | SteelSeries Sonar Virtual Audio Device`
	exe := `C:\Users\u\AppData\Local\Discord\app-1.0.9265\Discord.exe`
	session := func(endpoint string, flow int32, pid uint32) discoverySession {
		return discoverySession{sessionFacts{endpoint, endpoint + "|instance", "discord.exe", flow, 1, pid}, exe, false}
	}
	render := endpointFacts{ID: "{0.0.0.00000000}.{mic-render}", Name: "SteelSeries Sonar - Microphone (SteelSeries Sonar Virtual Audio Device)", Identity: identity, Flow: 0}
	render.Sessions = []discoverySession{session(render.ID, 0, 42)}
	capture := endpointFacts{ID: "{0.0.1.00000000}.{mic-capture}", Name: render.Name, Identity: identity, Flow: 1}
	capture.Sessions = []discoverySession{session(capture.ID, 1, 42)}
	headset := endpointFacts{ID: "{0.0.0.00000000}.{headset}", Name: "Headphones (Arctis Nova Pro Wireless)", Identity: `USB\VID_1038`, Flow: 0}
	headset.Sessions = []discoverySession{session(headset.ID, 0, 42)}
	return []endpointFacts{render, capture, headset}, []string{headset.ID, headset.ID, headset.ID}
}

func TestDiscoverEchoRoute(t *testing.T) {
	eps, defaults := discoveryFixture()
	route, reason := discoverEchoRoute(eps, defaults, "now")
	if route == nil {
		t.Fatal(reason)
	}
	if route.RenderID != eps[0].ID || route.CaptureID != eps[1].ID || route.Source != "auto" || route.ConfirmedAt != "now" || route.ExecutablePath != eps[0].Sessions[0].ExecutablePath {
		t.Fatalf("unexpected route %+v", route)
	}
	// The discovered route must satisfy the unchanged confirmed-route guard.
	f := echoFacts{Render: eps[0].Sessions[0].sessionFacts, Capture: eps[1].Sessions[0].sessionFacts, RenderIdentity: eps[0].Identity, CaptureIdentity: eps[1].Identity, ExecutablePath: route.ExecutablePath, IndependentPlayback: true}
	if ok, reason := confirmedEchoDecision(route, f); !ok {
		t.Fatal(reason)
	}
	// A render session that is already muted is still the same route.
	eps[0].Sessions[0].Muted = true
	if route, reason := discoverEchoRoute(eps, defaults, "now"); route == nil {
		t.Fatal(reason)
	}
}

func TestDiscoverEchoRouteRefusals(t *testing.T) {
	for name, alter := range map[string]func([]endpointFacts, []string) ([]endpointFacts, []string){
		"two candidates": func(e []endpointFacts, d []string) ([]endpointFacts, []string) {
			r, c := e[0], e[1]
			r.ID, c.ID = "{0.0.0.00000000}.{mic-render-2}", "{0.0.1.00000000}.{mic-capture-2}"
			r.Identity, c.Identity = r.Identity+" 2", c.Identity+" 2"
			return append(e, r, c), d
		},
		"two discord render sessions": func(e []endpointFacts, d []string) ([]endpointFacts, []string) {
			s := e[0].Sessions[0]
			s.Instance += "-2"
			e[0].Sessions = append(e[0].Sessions, s)
			return e, d
		},
		"capture session muted": func(e []endpointFacts, d []string) ([]endpointFacts, []string) {
			e[1].Sessions[0].Muted = true
			return e, d
		},
		"capture endpoint muted": func(e []endpointFacts, d []string) ([]endpointFacts, []string) { e[1].Muted = true; return e, d },
		"pid mismatch":           func(e []endpointFacts, d []string) ([]endpointFacts, []string) { e[1].Sessions[0].PID = 7; return e, d },
		"exe path mismatch": func(e []endpointFacts, d []string) ([]endpointFacts, []string) {
			e[1].Sessions[0].ExecutablePath = `C:\Other\Discord.exe`
			return e, d
		},
		"render is default":       func(e []endpointFacts, d []string) ([]endpointFacts, []string) { d[2] = e[0].ID; return e, d },
		"default unknown":         func(e []endpointFacts, d []string) ([]endpointFacts, []string) { d[0] = ""; return e, d },
		"no independent playback": func(e []endpointFacts, d []string) ([]endpointFacts, []string) { e[2].Sessions = nil; return e, d },
		"independent output muted": func(e []endpointFacts, d []string) ([]endpointFacts, []string) {
			e[2].Sessions[0].Muted = true
			return e, d
		},
		"wrong hardware identity": func(e []endpointFacts, d []string) ([]endpointFacts, []string) {
			e[0].Identity, e[1].Identity = "USB microphone", "USB microphone"
			return e, d
		},
		"capture identity differs": func(e []endpointFacts, d []string) ([]endpointFacts, []string) {
			e[1].Identity += " other"
			return e, d
		},
		"capture only": func(e []endpointFacts, d []string) ([]endpointFacts, []string) { e[0].Sessions = nil; return e, d },
		"non-discord": func(e []endpointFacts, d []string) ([]endpointFacts, []string) {
			for i := range e {
				for j := range e[i].Sessions {
					e[i].Sessions[j].Executable = "chrome.exe"
				}
			}
			return e, d
		},
		"discord name with other binary": func(e []endpointFacts, d []string) ([]endpointFacts, []string) {
			for i := range e {
				for j := range e[i].Sessions {
					e[i].Sessions[j].ExecutablePath = `C:\Tools\fake.exe`
				}
			}
			return e, d
		},
		"inactive render": func(e []endpointFacts, d []string) ([]endpointFacts, []string) {
			e[0].Sessions[0].State = 0
			return e, d
		},
		"inactive capture": func(e []endpointFacts, d []string) ([]endpointFacts, []string) {
			e[1].Sessions[0].State = 2
			return e, d
		},
		"render on capture id prefix": func(e []endpointFacts, d []string) ([]endpointFacts, []string) {
			e[0].ID = "{0.0.1.00000000}.{mic-render}"
			return e, d
		},
		"not sonar microphone": func(e []endpointFacts, d []string) ([]endpointFacts, []string) {
			e[0].Name, e[1].Name = "SteelSeries Sonar - Chat", "SteelSeries Sonar - Chat"
			return e, d
		},
	} {
		eps, defaults := discoveryFixture()
		eps, defaults = alter(eps, defaults)
		if route, _ := discoverEchoRoute(eps, defaults, "now"); route != nil {
			t.Errorf("%s: route discovered %+v", name, route)
		}
	}
}

func TestRediscoveryOnlyForStaleRoutes(t *testing.T) {
	for _, r := range []string{"endpoint_changed_reconfirm_route", "hardware_changed_reconfirm_route", "discord_binary_changed_reconfirm_route"} {
		if !routeRediscoverable(r) {
			t.Errorf("%s should allow rediscovery", r)
		}
	}
	for _, r := range []string{"waiting_for_active_unmuted_capture", "independent_discord_output_not_verified", "ambiguous_process_or_session", "capture_or_unknown_direction_protected", "confirmed_duplicate_render_only"} {
		if routeRediscoverable(r) {
			t.Errorf("%s must not replace the stored route", r)
		}
	}
}

func TestStartupApprovedFlag(t *testing.T) {
	for _, c := range []struct {
		data     []byte
		disabled bool
	}{{nil, false}, {[]byte{2, 0, 0}, false}, {[]byte{6}, false}, {[]byte{3, 0, 0}, true}, {[]byte{7}, true}} {
		if got := startupApprovedDisabled(c.data); got != c.disabled {
			t.Errorf("%v: got %v", c.data, got)
		}
	}
}

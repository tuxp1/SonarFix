package main

import (
	"path/filepath"
	"strings"
)

// Core Audio exposes a process and endpoint, not the semantic purpose of a
// Discord stream. Even a render stream can be communication audio. A separate
// user-confirmed route proof is therefore required before any targeted mute.
// Keep this policy pure, fail closed, and test it separately from COM.
type sessionFacts struct {
	Endpoint, Instance, Executable string
	Flow, State                    int32
	PID                            uint32
}

func muteDecision(s sessionFacts) (bool, string) {
	if s.Flow != 0 {
		return false, "protected_capture_or_unknown_direction"
	}
	if s.Endpoint == "" || s.Instance == "" || s.PID == 0 || s.State != 1 {
		return false, "incomplete_or_inactive_session"
	}
	if !strings.EqualFold(s.Executable, "discord.exe") {
		return false, "not_discord"
	}
	return false, "duplicate_route_not_proven_no_audio_muted"
}

func canonicalSonarOutput(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.TrimSuffix(n, " (steelseries sonar virtual audio device)")
	switch n {
	case "steelseries sonar - gaming", "steelseries sonar - game", "steelseries sonar - chat":
		return n
	}
	return ""
}

func sonarMic(name string) bool {
	lower := strings.ToLower(name)
	// Never match a speaker, Chat, Game, Stream, Media or a physical microphone.
	return strings.Contains(lower, "sonar") && strings.Contains(lower, "microphone")
}

// Source is "auto" for a discovered route; manual confirmations leave it empty.
type echoRoute struct {
	RenderID, CaptureID, HardwareIdentity, ExecutablePath, ConfirmedAt string
	Source                                                             string `json:",omitempty"`
}

// Inputs for automatic route discovery, read from Core Audio by the worker.
type discoverySession struct {
	sessionFacts
	ExecutablePath string
	Muted          bool
}
type endpointFacts struct {
	ID, Name, Identity string
	Flow               int32
	Muted              bool
	Sessions           []discoverySession
}

func activeDiscord(s discoverySession) bool {
	return s.State == 1 && s.PID != 0 && s.Instance != "" && strings.EqualFold(s.Executable, "discord.exe") && strings.EqualFold(filepath.Base(s.ExecutablePath), "discord.exe")
}
func sonarMicEndpoint(e endpointFacts, flow int32, prefix string) bool {
	return e.Flow == flow && strings.HasPrefix(e.ID, prefix) && sonarMic(e.Name) && strings.Contains(strings.ToLower(e.Identity), "steelseries sonar virtual audio device")
}

// discoverEchoRoute finds the duplicate route the manual mixer workaround
// proved: Discord playing into the Sonar Microphone RENDER endpoint while the
// same Discord process captures from the Sonar Microphone and also plays on a
// normal output. It only proposes a route; the caller still runs
// confirmedEchoDecision and the guarded mute. Any ambiguity returns nil.
func discoverEchoRoute(endpoints []endpointFacts, defaults []string, now string) (*echoRoute, string) {
	for _, id := range defaults {
		if id == "" {
			return nil, "default_output_unknown"
		}
	}
	var found []*echoRoute
	reason := "no_discord_playback_on_sonar_microphone"
	for _, r := range endpoints {
		if !sonarMicEndpoint(r, 0, "{0.0.0.") {
			continue
		}
		var renders []discoverySession
		for _, s := range r.Sessions {
			if activeDiscord(s) {
				renders = append(renders, s)
			}
		}
		if len(renders) == 0 {
			continue
		}
		if len(renders) > 1 {
			return nil, "ambiguous_discord_render_sessions"
		}
		render := renders[0]
		for _, id := range defaults {
			if strings.EqualFold(id, r.ID) {
				return nil, "sonar_microphone_is_default_output"
			}
		}
		var captureIDs []string
		reason = "no_matching_active_unmuted_capture"
		for _, c := range endpoints {
			if !sonarMicEndpoint(c, 1, "{0.0.1.") || c.Identity != r.Identity || c.Muted {
				continue
			}
			for _, s := range c.Sessions {
				if activeDiscord(s) && !s.Muted && s.PID == render.PID && strings.EqualFold(s.ExecutablePath, render.ExecutablePath) {
					captureIDs = append(captureIDs, c.ID)
					break
				}
			}
		}
		if len(captureIDs) != 1 {
			if len(captureIDs) > 1 {
				return nil, "ambiguous_capture_endpoints"
			}
			continue
		}
		independent := false
		for _, o := range endpoints {
			if o.Flow != 0 || o.ID == r.ID || !strings.HasPrefix(o.ID, "{0.0.0.") {
				continue
			}
			for _, s := range o.Sessions {
				if activeDiscord(s) && !s.Muted && s.PID == render.PID && strings.EqualFold(s.ExecutablePath, render.ExecutablePath) {
					independent = true
				}
			}
		}
		if !independent {
			reason = "independent_discord_output_not_verified"
			continue
		}
		found = append(found, &echoRoute{RenderID: r.ID, CaptureID: captureIDs[0], HardwareIdentity: r.Identity, ExecutablePath: render.ExecutablePath, ConfirmedAt: now, Source: "auto"})
	}
	if len(found) > 1 {
		return nil, "multiple_route_candidates"
	}
	if len(found) == 0 {
		return nil, reason
	}
	return found[0], "auto_detected"
}

// Reasons where a stored route no longer matches this PC but a fresh,
// fully guarded discovery may replace it (Discord updates change its path).
func routeRediscoverable(reason string) bool {
	switch reason {
	case "endpoint_changed_reconfirm_route", "hardware_changed_reconfirm_route", "discord_binary_changed_reconfirm_route":
		return true
	}
	return false
}

type echoFacts struct {
	Render, Capture                                    sessionFacts
	RenderIdentity, CaptureIdentity, ExecutablePath    string
	CaptureMuted, IndependentPlayback, TargetIsDefault bool
}

func confirmedEchoDecision(route *echoRoute, f echoFacts) (bool, string) {
	if route == nil || route.ConfirmedAt == "" {
		return false, "echo_route_not_confirmed"
	}
	if f.Render.Flow != 0 || f.Capture.Flow != 1 {
		return false, "capture_or_unknown_direction_protected"
	}
	if route.RenderID == "" || route.CaptureID == "" || !strings.HasPrefix(route.RenderID, "{0.0.0.") || !strings.HasPrefix(route.CaptureID, "{0.0.1.") {
		return false, "invalid_route_ids"
	}
	if f.Render.Endpoint != route.RenderID || f.Capture.Endpoint != route.CaptureID {
		return false, "endpoint_changed_reconfirm_route"
	}
	if route.HardwareIdentity == "" || f.RenderIdentity != route.HardwareIdentity || f.CaptureIdentity != route.HardwareIdentity {
		return false, "hardware_changed_reconfirm_route"
	}
	if !strings.Contains(strings.ToLower(route.HardwareIdentity), "steelseries sonar virtual audio device") {
		return false, "unverified_hardware"
	}
	if route.ExecutablePath == "" || !strings.EqualFold(f.ExecutablePath, route.ExecutablePath) {
		return false, "discord_binary_changed_reconfirm_route"
	}
	if f.Render.Instance == "" || f.Capture.Instance == "" || f.Render.PID == 0 || f.Render.PID != f.Capture.PID || f.Render.Executable != "discord.exe" || f.Capture.Executable != "discord.exe" {
		return false, "ambiguous_process_or_session"
	}
	if f.Render.State != 1 || f.Capture.State != 1 || f.CaptureMuted {
		return false, "waiting_for_active_unmuted_capture"
	}
	if f.TargetIsDefault || !f.IndependentPlayback {
		return false, "independent_discord_output_not_verified"
	}
	return true, "confirmed_duplicate_render_only"
}

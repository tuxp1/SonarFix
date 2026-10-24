//go:build windows

package main

import "testing"

// The status line may only say "waiting" when the Sonar Microphone exists.
func TestSonarMicrophonePresent(t *testing.T) {
	endpoints, _ := discoveryFixture()
	if !sonarMicrophonePresent(endpoints) {
		t.Fatal("both Sonar Microphone endpoints are present")
	}
	if sonarMicrophonePresent(endpoints[1:]) {
		t.Fatal("missing playback endpoint must report unavailable")
	}
	if sonarMicrophonePresent([]endpointFacts{endpoints[0], endpoints[2]}) {
		t.Fatal("missing recording endpoint must report unavailable")
	}
	if sonarMicrophonePresent(nil) {
		t.Fatal("no endpoints must report unavailable")
	}
}

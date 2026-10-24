# SonarFix v1.0.0 — Test Report

Date: 2026-10-24. Clean build (`dist/`, `reports/`, generated resources and Go test cache removed first). Windows 11, 3840×2160 at 200% scaling, SteelSeries GG/Sonar with an Arctis headset.

## Stability review

The worker, settings app, installer, update checker and translations were reviewed before the version change. No blocking defect was found, so no functional code changed for v1.0.0; only the version number and documentation did. One existing behaviour is documented as a limitation: with the tray icon on, the installer asks the user to exit SonarFix before updating.

## Automated (all passing)

| Suite | Result |
| --- | --- |
| Go tests (`build.ps1`) | 21 pass, 3 skipped (live-audio tests that only run with explicit consent). Covers route discovery and refusals, the rule that only a confirmed duplicate playback session may be muted, ownership/recovery journals, Game & Chat targeting only the two Sonar outputs, startup flag, and the "device unavailable" status. |
| `go vet` | Passes; the 4 known `unsafeptr` notes on raw COM pointers remain. |
| Settings self-test | Language switching with RTL/LTR; dialogs relocalize, Cancel never confirms, main window stays open behind them; layout at logical 100/125/150% with Advanced open and closed; Save only with unsaved changes; language button in the header; status lines; tray default OFF and no audio commands from tray toggles; fixed update source; update checker against offline, rate-limited, malformed, draft and pre-release fixtures. |
| `test-ui.ps1` | Self-test plus the installer in English and Arabic: in-place language switches keep all choices, output section follows Game & Chat, mirrored layout fits. Audio settings, UI settings, startup, language and worker PID unchanged afterwards. |
| `test-release.ps1` | Version 1.0.0 / 1.0.0.0 in all executables and the installer; the four core audio files match their baseline hashes; installer keeps the restore-before-uninstall hooks. |
| `package-source.ps1` | Allowlisted files only; secret and private-path scan clean. |
| `verify-release.ps1` | Installer and source ZIP match `SHA256.json`; installer hash in the release notes; source ZIP byte-identical to the tree; GitHub files present; license, notices and fixed update source verified. |

## Manual on this PC (non-destructive)

| Check | Result |
| --- | --- |
| Update 0.2 → 1.0.0 (silent) | Exit 0. Apps list shows "SonarFix v1.0.0". Audio settings file byte-identical; tray and update preferences kept; installed executables identical to the build. With the tray host running, the silent update first stopped at "SonarFix is already running"; after exiting the tray host it completed. |
| Single worker and startup | One `SonarFix.exe` and one settings/tray process. One `Run` entry, enabled; no Startup-folder shortcut or scheduled task. |
| Read-only health check after update | Ready: Sonar present, Game and Chat disabled, Sonar Microphone active (not muted or disabled by SonarFix), output active, worker running. |
| Tray off/on | Turning the tray off and on saved immediately; the worker kept the same process ID; still one settings process. |
| Uninstall, then Cancel | SonarFix's confirmation appeared over the main window; after Cancel the files, registration (1.0.0) and running worker were unchanged. |
| Save Changes | Off with no changes, on after a change, off after undoing it (checked earlier on this build line, and by the self-test on this build). |
| Arabic text | All 142 Arabic strings rendered with the app's rules and reviewed. |
| Update check | Before publishing, GitHub returned 404 for the release and the app showed "Couldn't check for updates right now. Audio features are not affected."; the worker kept running. |

Earlier evidence from the same audio engine: `SonarFix.log` on this PC records, on 2026-10-23, an automatically detected duplicate route, the Discord playback session muted with the microphone capture "preserved", and the mute restored afterwards. That predates this release and is not counted as a test of it.

## Not tested

- An audible Discord screen-share check of v1.0.0, and ClearCast AI behaviour during it.
- Startup after a real Windows sign-in, and the tray appearing at sign-in.
- A clean install on a PC that never had SonarFix (no Windows Sandbox here). Tray default OFF is covered by the self-test.
- A completed uninstall and Restore Audio Settings on the live setup; both would change the working audio configuration.
- Physical 125% and 150% Windows scaling.

Machine-specific logs are in the ignored `reports/` folder and are not published.

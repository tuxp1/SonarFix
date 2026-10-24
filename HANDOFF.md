# SonarFix — Handoff

## Status

**SonarFix v1.0.0 is the first official stable release.** Repository: https://github.com/tuxp1/SonarFix (public, `main`). Release: https://github.com/tuxp1/SonarFix/releases/tag/v1.0.0 with the installer, source ZIP and SHA256.json. License: MIT, `Copyright (c) 2026 tuxp1`. Development of v1.0.0 is closed; fixes go into 1.0.x, new features into 1.1.0.

Earlier private builds were numbered 3.x and 0.2. Installing 1.0.0 over any of them keeps settings and recovery records; the recovery journal format did not change with the version number.

## Features

1. **Prevent Duplicate Mic Audio:** the worker finds Discord playing into the Sonar Microphone playback endpoint while it records from the Sonar Microphone and also plays on a normal output, then mutes only that Discord playback session. It never mutes the microphone input; any ambiguity means no action.
2. **Disable Sonar Game & Chat:** disables both Sonar outputs and re-checks every 3 s. The user picks a physical output and whether it becomes the Windows default.
3. **Audio Device Health Check:** read-only (`SonarFix.exe --health-info`).
4. **Optional System Tray:** off by default; Open, Stop, Exit. The tray host is the settings process; it never starts a second monitor.
5. **Start with Windows:** one `HKCU\...\Run` entry (`SonarFix.exe --background`), kept in step with Task Manager's StartupApproved flag.
6. **GitHub Update Checker:** see below.
7. **Arabic & English:** app and installer switch instantly; choice stored in `HKCU\Software\SonarFix\UILanguage`.
8. **Restore Audio Settings:** turns both features off and restores every journaled change.
9. **Diagnostics & Logs:** open the data folder or export a diagnostics JSON. Nothing is uploaded.
10. **Installer & Uninstaller:** per-user Inno Setup; uninstall restores audio first and aborts if that fails.

## Layout

| Path | Contents |
| --- | --- |
| `SonarFix-source/` | Go worker `SonarFix.exe`. Core audio files (`audio_windows.go`, `device_control.go`, `echo_windows.go`, `safety.go`) are pinned by `tests/audio-core-baseline.json`; change them only with a live audio test. |
| `SonarFix-source/installer_info_windows.go` | Read-only status report (`--setup-info`) used by the installer and the status line. |
| `manager/` | WPF settings window `SonarFix.Settings.exe` (C#, .NET Framework 4.8): main window, tray, health dialog, About/updates, self-tests. |
| `installer/` | `SonarFix.iss` (Inno Setup 7.1), `Arabic-LTR.isl`, logo PNGs. `Version.iss` and `Translations.iss` are generated. |
| `localization/strings.json` | Every English/Arabic string for app and installer. |
| `docs/` | `github-setup.md` (publishing steps), `releases/v<version>.md` (release notes with installer checksum). |
| `.github/` | Issue forms and PR template. |

Data: `%LOCALAPPDATA%\SonarFix` (`settings.json` audio options and recovery journal, `ui-settings.json` tray/update options, `status.json`, `SonarFix.log`). Install: `%LOCALAPPDATA%\Programs\SonarFix`.

## Engineering and safety decisions

- Audio code was not changed in the final polish. Status reporting gained one read-only check: `EchoState=device_unavailable` when either Sonar Microphone endpoint is missing, so the UI no longer says "waiting" with no microphone.
- Status line = worker state, never the toggle: stopped, off, waiting for screen sharing, ready, duplicate audio blocked (route confirmed and the session actually muted), route uncertain, device unavailable, status unavailable.
- Save Changes compares echo, Game & Chat, output, redirect and startup with the last loaded/saved values. Language and tray apply immediately and are not "unsaved changes". Nothing is applied to audio before Save.
- Arabic: `T()` (app) and `Txt()` (installer) isolate Latin names with LRM, keep "Game & Chat" as one LTR unit, and the installer only applies an RTL embedding to text that contains Arabic.
- Updates: the source is the constant `OfficialRepository = "tuxp1/SonarFix"` in `manager/Preferences.cs`. There is no UI, file or setting to change it. Checks are HTTPS to the GitHub Releases API, no redirects, 1 MB limit, stable releases only, and only open the release page. Failures show "Couldn't check for updates…" and never touch the worker. Automatic daily checks are off by default.
- Installer: `WizardStyle=modern dark hidebevels`, `WizardBackColor=#10151F`. Buttons, switches, checkboxes, radios and the A/ع button are painted `TBitmapButton`s (focusable, named for screen readers); native Back/Next/Cancel stay underneath so Enter/Esc work. An existing install offers Update (keeps settings, skips options), Modify (options; folder locked) or Uninstall. Update falls back to Modify if the saved output is gone.
- Validation switches (never install): `/VALIDATEONLY=1`, `/UIPAGE=maintenance|config`, `/UIFRESH=1`, `/UITESTFILE=<file>`.

## Build

Tools: Windows PowerShell 5.1 or 7, Go 1.23.2+, go-winres 0.3.3, Inno Setup 7.1, .NET Framework 4.8 `csc` (part of Windows). Put them on PATH or pass `-Go`, `-WinRes`, `-ISCC` to `build.ps1`; an ignored `.tools/` folder is also checked. If scripts are blocked: `powershell -ExecutionPolicy Bypass -File <script>`.

`build.ps1` generates version/translation files, runs Go tests and vet, builds both executables, runs the UI self-test and compiles `dist/SonarFix-Setup-<version>.exe`.

## Release procedure

1. Change `VERSION` (and only that) for a new version; add a `CHANGELOG.md` entry and `docs/releases/v<version>.md`.
2. Delete `dist/` and `reports/`.
3. `build.ps1`, `test-ui.ps1`, `test-release.ps1`.
4. Put the installer's SHA-256 (from `dist/SHA256.json`) into the release notes.
5. `package-source.ps1`, then `verify-release.ps1`. It must print PASS: installer = `SHA256.json` = release notes, source ZIP byte-identical to the tree, required GitHub files present.
6. Deliver only those `dist/` files. Every rebuild changes the installer hash; never reuse an old checksum.
7. Commit the verified tree to `main`, tag `v<version>`, and publish a stable (not draft, not pre-release) release with the installer, source ZIP and SHA256.json, per `docs/github-setup.md`. Download the published files and compare their SHA-256.

## Test status

See `TEST-REPORT.md`. All automated suites pass. Still needs a person: audible Discord screen-share check and ClearCast, real sign-in startup, tray at sign-in, a clean install on a PC that never had SonarFix, a full uninstall, and physical 125%/150% scaling.

## Remaining limitations

Unsigned builds; one tested headset setup; Discord may need its own output changed; with the tray on, the installer asks the user to exit SonarFix from the tray before updating (the settings mutex is held by the tray host); the four known `unsafeptr` vet notes on COM pointers remain.

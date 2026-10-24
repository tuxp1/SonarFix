<p align="center">
  <img src="installer/images/wizard-small-112.png" width="88" height="88" alt="SonarFix logo">
</p>

<h1 align="center">SonarFix</h1>

<p align="center">
  A lightweight Windows utility that prevents duplicate microphone audio in Discord and lets you disable unwanted SteelSeries Sonar audio devices.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/version-1.0.0-267B71" alt="Version 1.0.0">
  <img src="https://img.shields.io/badge/license-MIT-267B71" alt="MIT license">
  <img src="https://img.shields.io/badge/platform-Windows%2010%20%7C%2011%20x64-2B3A4F" alt="Windows 10 and 11, x64">
</p>

<p align="center">
  <b>English</b> · <a href="README.ar.md">العربية</a>
</p>

> SonarFix is an independent, unofficial community project and is not affiliated with, endorsed by, or sponsored by SteelSeries or Discord.

## Features

| Feature | What it does |
| --- | --- |
| **Prevent Duplicate Mic Audio** | When Discord plays your own voice back into the Sonar Microphone device during screen sharing, SonarFix mutes only that duplicate Discord playback stream. Your microphone input stays unmuted. |
| **Disable Sonar Game & Chat** | Turns off the Sonar Game and Chat output devices you don't use and keeps them off. You choose a physical output (for example your headset), and you decide whether it becomes the Windows default. |
| **Audio Device Health Check** | A read-only check that shows whether Sonar, Game, Chat, the Sonar Microphone and your chosen output are in the expected state. It never changes a device. |
| **Optional System Tray** | An icon with Open, Stop and Exit. It is off by default; audio monitoring does not depend on it. |
| **Update Checker** | Checks this repository's GitHub Releases for a newer stable version when you ask (or once a day if you opt in). It only shows the release; it never downloads or installs anything, and a failed check never affects audio. |
| **Start with Windows** | Optional. Starts the background worker silently when you sign in. |
| **Arabic & English** | The app and the installer switch language instantly with the A/ع button, with full right-to-left layout. |
| **Restore Audio Settings** | Turns both audio features off and puts back every audio change SonarFix recorded. |
| **Diagnostics & Logs** | Opens the log folder, or exports a diagnostics file you can attach to a bug report. Nothing is sent anywhere. |
| **Installer & Uninstaller** | Per-user installer with Update, Modify and Uninstall. Uninstalling restores recorded audio changes before removing files. |

## Download

Download the latest `SonarFix-Setup-x.y.exe` from the [**Releases**](https://github.com/tuxp1/SonarFix/releases/latest) page. Each release lists the SHA-256 checksum of the installer so you can verify the download:

```powershell
Get-FileHash .\SonarFix-Setup-1.0.0.exe -Algorithm SHA256
```

### Requirements

- Windows 10 or 11, 64-bit
- .NET Framework 4.8 (included with current Windows versions)
- SteelSeries GG with Sonar
- Discord, for the duplicate-mic feature

## Installation

1. Run `SonarFix-Setup-1.0.0.exe`. No administrator rights are needed; SonarFix installs for your user only.
2. Choose the features you want. If you turn on **Disable Game & Chat**, pick the output device you actually listen on.
3. Click **Install**. SonarFix starts in the background and, if you chose it, starts with Windows.

If SonarFix is already installed, the installer offers **Update** (keep your settings), **Modify** (review features first) or **Uninstall**.

The installer and app are not code-signed yet, so Windows SmartScreen may show a warning the first time. See [Troubleshooting](#troubleshooting).

## Uninstall

Use **Advanced Settings → Uninstall SonarFix**, or *Settings → Apps → Installed apps → SonarFix → Uninstall* in Windows. SonarFix asks for confirmation, restores the audio changes it recorded, and then removes its files. If restoring fails, nothing is removed. Your preferences in `%LOCALAPPDATA%\SonarFix` are kept.

## Known limitations

- Builds are not code-signed.
- Duplicate-audio detection only acts when the Discord audio route is unambiguous; otherwise it does nothing and logs why.
- Discord may need its own output device set after Game & Chat are disabled.
- With the tray icon on, the installer asks you to close SonarFix before updating. Choose **Exit** from the tray icon, then run the installer again.
- Tested with one SteelSeries headset setup on Windows 11; other headsets and SteelSeries GG versions have not been verified.

## How it works

SonarFix has two parts: a small background worker (`SonarFix.exe`) and the settings window (`SonarFix.Settings.exe`).

- **Duplicate mic audio.** During screen sharing, Discord can end up playing audio into the Sonar Microphone device while it is also recording from it, so people hear you twice. The worker looks for that exact pattern using the Windows audio APIs. It acts only when the route is unambiguous, and then mutes that single Discord playback stream. If anything is uncertain, it does nothing.
- **Game & Chat.** The worker disables the Sonar Game and Chat output devices and checks every few seconds that they stay disabled, since Sonar can turn them back on. Your selected physical output keeps working.
- **Safety.** Every change is saved to a recovery journal under `%LOCALAPPDATA%\SonarFix`. Stop, Restore and Uninstall put back what SonarFix changed.

SonarFix does not capture, record or upload audio, and it does not send diagnostics anywhere. The only network request is the optional update check to the GitHub Releases API of the official repository, `tuxp1/SonarFix`.

## Screenshots

Screenshots of the app and the installer will be added here.

<!--
Add real screenshots to docs/screenshots/ and reference them here, for example:
![SonarFix settings](docs/screenshots/settings.png)
![SonarFix installer](docs/screenshots/installer.png)
-->

## Troubleshooting

<details>
<summary><b>Discord still echoes</b></summary>

- Make sure **Prevent Duplicate Mic Audio** is on and the status at the top says SonarFix is running.
- The fix applies while you are screen sharing. The status line shows whether SonarFix is waiting, ready or unsure about the audio route.
- Open **Advanced Settings → Open Logs** and check `SonarFix.log` for the reason. If the route is reported as uncertain, SonarFix deliberately does not mute anything.
</details>

<details>
<summary><b>Microphone is not detected</b></summary>

- Run **Advanced Settings → Check Audio Devices**. It shows whether the Sonar Microphone is available.
- Make sure SteelSeries GG is running and Sonar is enabled.
- SonarFix never mutes or disables your microphone input. If the microphone is muted, check Windows Sound settings and SteelSeries GG.
</details>

<details>
<summary><b>Game or Chat becomes enabled again</b></summary>

- **Disable Game & Chat** must be on and SonarFix must be running; Stop pauses enforcement.
- If you turned off **Start with Windows**, open SonarFix after signing in.
- **Check Audio Devices** shows the current state. If it reports a mismatch, the background worker retries automatically.
</details>

<details>
<summary><b>Audio output is unavailable</b></summary>

- The output you selected may be unplugged or renamed. Choose another output under **Disable Game & Chat** and click **Save Changes**.
- To undo everything SonarFix changed, use **Advanced Settings → Restore Audio Settings**.
</details>

<details>
<summary><b>Windows or antivirus warnings</b></summary>

- The builds are not code-signed yet, so SmartScreen may say the publisher is unknown. Download only from this repository's Releases page and compare the SHA-256 checksum with the one in the release notes.
- If your antivirus flags SonarFix, please don't disable your protection. Report a false positive to your antivirus vendor and [open an issue](https://github.com/tuxp1/SonarFix/issues) with the detection name.
- You can also build SonarFix yourself from source (see below).
</details>

## Build from source

### Requirements

| Tool | Version | Purpose |
| --- | --- | --- |
| Windows PowerShell 5.1 or PowerShell 7 | | Build scripts |
| [Go](https://go.dev/dl/) | 1.23.2 or newer | Background worker |
| [go-winres](https://github.com/tc-hib/go-winres) | 0.3.3 | Worker icon and version resources |
| [Inno Setup](https://jrsoftware.org/isinfo.php) | 7.1 | Installer |
| .NET Framework 4.8 developer tools | | `csc.exe` and WPF, already part of Windows |

No NuGet packages or other third-party libraries are used.

```powershell
go install github.com/tc-hib/go-winres@v0.3.3
```

### Build

Put `go`, `go-winres` and `ISCC` on your `PATH`, or pass their paths:

```powershell
./build.ps1 -Go C:\Tools\Go\bin\go.exe -WinRes C:\Tools\go-winres.exe -ISCC "C:\Program Files (x86)\Inno Setup 7\ISCC.exe"
```

`build.ps1` generates version and translation files, runs the Go tests and `go vet`, builds `SonarFix.exe` and `SonarFix.Settings.exe`, runs the UI self-test and compiles the installer. Everything goes to `dist/`:

| Output | Description |
| --- | --- |
| `dist/SonarFix.exe` | Background worker |
| `dist/SonarFix.Settings.exe` | Settings window |
| `dist/SonarFix-Setup-<version>.exe` | Installer |
| `dist/SHA256.json` | Checksums of the files above |

Then run the checks and create the source archive:

```powershell
./test-ui.ps1         # UI and installer layout tests, read-only
./test-release.ps1    # version metadata and installer integration checks
./package-source.ps1  # dist/SonarFix-<version>-source.zip
./verify-release.ps1  # installer checksum, release notes and source ZIP all match
```

If script execution is disabled on your PC, run a script with `powershell -ExecutionPolicy Bypass -File .\build.ps1`.

### Project layout

| Path | Contents |
| --- | --- |
| `SonarFix-source/` | Go worker: audio monitoring, recovery journals, tests |
| `manager/` | Settings window (WPF, C#) and its self-tests |
| `installer/` | Inno Setup script, Arabic messages and logo images |
| `localization/strings.json` | All English and Arabic text, shared by the app and the installer |
| `VERSION` | The single version number used everywhere |

## Contributing

Bug reports, translations and pull requests are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) first. To report a security issue, follow [SECURITY.md](SECURITY.md).

## License

SonarFix is released under the [MIT License](LICENSE). Third-party components keep their own licenses; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) and [LICENSES/](LICENSES).

SteelSeries, Sonar, SteelSeries GG and Discord are trademarks of their respective owners and are used here only to describe compatibility.

# User guide

[العربية](user-guide.ar.md) · [Back to README](../README.md)

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

Every audio change SonarFix makes is recorded so it can be undone with **Restore Audio Settings** or by uninstalling.

## Verify the download

Each release lists the SHA-256 checksum of the installer:

```powershell
Get-FileHash .\SonarFix-Setup-1.0.0.exe -Algorithm SHA256
```

## Installation

1. Run `SonarFix-Setup-1.0.0.exe`. No administrator rights are needed; SonarFix installs for your user only.
2. Choose the features you want. If you turn on **Disable Game & Chat**, pick the output device you actually listen on.
3. Click **Install**. SonarFix starts in the background and, if you chose it, starts with Windows.

If SonarFix is already installed, the installer offers **Update** (keep your settings), **Modify** (review features first) or **Uninstall**.

The installer and app are not code-signed yet, so Windows SmartScreen may show a warning the first time. See [Troubleshooting](troubleshooting.md#windows-or-antivirus-warnings).

## Uninstall

Use **Advanced Settings → Uninstall SonarFix**, or *Settings → Apps → Installed apps → SonarFix → Uninstall* in Windows. SonarFix asks for confirmation, restores the audio changes it recorded, and then removes its files. If restoring fails, nothing is removed. Your preferences in `%LOCALAPPDATA%\SonarFix` are kept.

## Known limitations

- Builds are not code-signed.
- Duplicate-audio detection only acts when the Discord audio route is unambiguous; otherwise it does nothing and logs why.
- Discord may need its own output device set after Game & Chat are disabled.
- With the tray icon on, the installer asks you to close SonarFix before updating. Choose **Exit** from the tray icon, then run the installer again.
- Tested with one SteelSeries headset setup on Windows 11; other headsets and SteelSeries GG versions have not been verified.

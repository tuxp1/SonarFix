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

- **Prevent duplicate mic audio in Discord** — mutes only Discord's duplicate playback during screen sharing; your microphone is never muted.
- **Disable Sonar Game & Chat** — turns off the Sonar outputs you don't use and keeps them off.
- **Restore and check your audio** — undo every change SonarFix made, and see the state of your Sonar devices at a glance.

Also: Arabic & English, optional tray icon, start with Windows and an update checker. See the [user guide](docs/user-guide.md) for the full list.

## Screenshots

### English

<p align="center">
  <img src="docs/screenshots/english.png" width="720" alt="SonarFix settings window in English">
</p>

### Arabic

<p align="center">
  <img src="docs/screenshots/arabic.png" width="720" alt="SonarFix settings window in Arabic">
</p>

## Download

**[Download the latest release](https://github.com/tuxp1/SonarFix/releases/latest)** (`SonarFix-Setup-x.y.exe`).

**Requirements:** Windows 10 or 11 (64-bit), .NET Framework 4.8 (included with Windows), SteelSeries GG with Sonar, and Discord for the duplicate-mic feature.

**Install:** run the installer (no administrator rights needed), choose your features and, if you disable Game & Chat, the output you listen on. Installing over an existing copy offers Update, Modify or Uninstall.

> [!NOTE]
> The installer is not code-signed yet, so Windows SmartScreen may warn on first run. Each release lists the installer's SHA-256 checksum so you can verify it. SonarFix has been tested with one SteelSeries headset setup on Windows 11; see [known limitations](docs/user-guide.md#known-limitations).

## Documentation

- [User guide](docs/user-guide.md) — all features, installation, uninstall and known limitations
- [How it works](docs/how-it-works.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Build from source](docs/building.md)

## Contributing

Bug reports, translations and pull requests are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) first. To report a security issue, follow [SECURITY.md](SECURITY.md).

## License

SonarFix is released under the [MIT License](LICENSE). Third-party components keep their own licenses; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) and [LICENSES/](LICENSES).

SteelSeries, Sonar, SteelSeries GG and Discord are trademarks of their respective owners and are used here only to describe compatibility.

# Changelog

All notable changes to SonarFix are listed here. Versions follow [Semantic Versioning](https://semver.org/); two-part tags such as `v1.0` mean `1.0.0`.

## v1.0.0 — 2026-10-24, first stable release

### Features

- **Prevent Duplicate Mic Audio:** detects the duplicate Discord playback route into the Sonar Microphone during screen sharing and mutes only that stream. The microphone input is never muted; ambiguous routes are left alone.
- **Disable Game & Chat:** disables the Sonar Game and Chat outputs and keeps them disabled, with an explicit choice of physical output and of the Windows default.
- **Audio Device Health Check:** read-only status of Sonar, Game, Chat, the Sonar Microphone and the selected output.
- **Optional system tray**, off by default, with Open, Stop and Exit.
- **Update checker** for stable releases of the official repository `tuxp1/SonarFix`, on demand or daily if enabled. It never downloads or installs anything. About shows the version, Check for Updates, the daily option and a GitHub button; the update source cannot be changed from the UI.
- **Arabic and English** in the app and the installer, switched instantly with the A/ع button.
- Recovery journal for every audio change, used by Restore Audio Settings, Stop and Uninstall.

### Installer

- Redesigned to match the app: dark theme, centered logo, turquoise switches and checkboxes, and consistent Back/Install/Cancel buttons.
- An existing installation now offers Update, Modify or Uninstall on its own page instead of a red button on the welcome screen.
- The output device appears only when Disable Game & Chat is on. Choices are kept when switching language.
- Shorter window; the install path is no longer pre-selected.

### Settings window

- Compact Advanced Settings: Startup, Audio & Diagnostics (two-column actions), Application and a separate Uninstall section.
- The window grows with its content up to the screen height, with a slim dark scrollbar only when needed.
- Clearer Check Audio Devices result, with one row per device.
- Version shown only in About SonarFix.
- The A/ع language button sits at the top of the window instead of inside the status card.
- Save Changes is enabled only while there are unsaved audio changes, and turns off again after saving or when the changes are undone.
- The status line follows what the worker actually reports: waiting for screen sharing, ready, duplicate audio blocked, stopped, or audio device unavailable when the Sonar Microphone is missing.
- Arabic text keeps "Game & Chat" in its English order, and Latin-only captions in the installer are no longer reversed.

### Build

- Generated version files are written as UTF-8 without BOM, so the build works in Windows PowerShell 5.1 as well as PowerShell 7.

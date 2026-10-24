# Contributing to SonarFix

Thanks for helping. SonarFix is small on purpose, so the most useful contributions are bug reports with logs, fixes, and translation improvements.

## Reporting a bug

Use the **Bug report** issue form. Please include:

- Windows version, SonarFix version (shown in **About SonarFix**) and SteelSeries GG version.
- What you expected and what happened.
- The relevant lines from `%LOCALAPPDATA%\SonarFix\SonarFix.log`, or a diagnostics file from **Advanced Settings → Export Diagnostics**.

Diagnostics and logs contain audio device names and IDs from your PC. Look through them before attaching, and remove anything you don't want to share.

## Before you open a pull request

1. Open an issue first for anything larger than a small fix, so we can agree on the approach.
2. Keep changes focused. Don't mix refactoring with behavior changes.
3. Build and test locally:

   ```powershell
   ./build.ps1
   ./test-ui.ps1
   ./test-release.ps1
   ```

4. Describe what you tested on a real PC and what you could not test (for example, live Discord screen sharing).

## Ground rules for audio code

The worker changes real audio devices on people's PCs, so changes there are held to a higher bar:

- Never mute or disable the user's microphone input.
- Fail closed: if the audio route is ambiguous, do nothing and log why.
- Record every change in the recovery journal so Restore and Uninstall can undo it.
- `tests/audio-core-baseline.json` pins the hashes of the core audio files. `test-release.ps1` fails when they change. If your pull request changes them on purpose, update the baseline in the same pull request and explain why.

## Translations

All text lives in `localization/strings.json`, with an `en` and an `ar` value for each key. The installer translations in `installer/Translations.iss` are generated from it by `generate-translations.ps1` during the build; don't edit that file by hand.

When adding Arabic text, keep product names (SonarFix, Discord, Windows, Game, Chat) in Latin letters. The app and installer isolate them automatically so mixed text displays in the right order.

## Code style

- Match the style of the file you are editing.
- No new third-party dependencies or NuGet packages without discussion first.
- Generated files (`manager/Version.cs`, `SonarFix-source/version.go`, `installer/Version.iss`, `installer/Translations.iss`) come from `VERSION` and `strings.json`. Change the source, then rebuild.

## License

By contributing, you agree that your contribution is licensed under the [MIT License](LICENSE).

# How SonarFix works

[العربية](how-it-works.ar.md) · [Back to README](../README.md)

SonarFix has two parts: a small background worker (`SonarFix.exe`) and the settings window (`SonarFix.Settings.exe`).

- **Duplicate mic audio.** During screen sharing, Discord can end up playing audio into the Sonar Microphone device while it is also recording from it, so people hear you twice. The worker looks for that exact pattern using the Windows audio APIs. It acts only when the route is unambiguous, and then mutes that single Discord playback stream. If anything is uncertain, it does nothing.
- **Game & Chat.** The worker disables the Sonar Game and Chat output devices and checks every few seconds that they stay disabled, since Sonar can turn them back on. Your selected physical output keeps working.
- **Safety.** Every change is saved to a recovery journal under `%LOCALAPPDATA%\SonarFix`. Stop, Restore and Uninstall put back what SonarFix changed.

SonarFix does not capture, record or upload audio, and it does not send diagnostics anywhere. The only network request is the optional update check to the GitHub Releases API of the official repository, `tuxp1/SonarFix`.

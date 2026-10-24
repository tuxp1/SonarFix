# Troubleshooting

[العربية](troubleshooting.ar.md) · [Back to README](../README.md)

## Discord still echoes

- Make sure **Prevent Duplicate Mic Audio** is on and the status at the top says SonarFix is running.
- The fix applies while you are screen sharing. The status line shows whether SonarFix is waiting, ready or unsure about the audio route.
- Open **Advanced Settings → Open Logs** and check `SonarFix.log` for the reason. If the route is reported as uncertain, SonarFix deliberately does not mute anything.

## Microphone is not detected

- Run **Advanced Settings → Check Audio Devices**. It shows whether the Sonar Microphone is available.
- Make sure SteelSeries GG is running and Sonar is enabled.
- SonarFix never mutes or disables your microphone input. If the microphone is muted, check Windows Sound settings and SteelSeries GG.

## Game or Chat becomes enabled again

- **Disable Game & Chat** must be on and SonarFix must be running; Stop pauses enforcement.
- If you turned off **Start with Windows**, open SonarFix after signing in.
- **Check Audio Devices** shows the current state. If it reports a mismatch, the background worker retries automatically.

## Audio output is unavailable

- The output you selected may be unplugged or renamed. Choose another output under **Disable Game & Chat** and click **Save Changes**.
- To undo everything SonarFix changed, use **Advanced Settings → Restore Audio Settings**.

## Windows or antivirus warnings

- The builds are not code-signed yet, so SmartScreen may say the publisher is unknown. Download only from this repository's Releases page and compare the SHA-256 checksum with the one in the release notes.
- If your antivirus flags SonarFix, please don't disable your protection. Report a false positive to your antivirus vendor and [open an issue](https://github.com/tuxp1/SonarFix/issues) with the detection name.
- You can also [build SonarFix yourself from source](building.md).

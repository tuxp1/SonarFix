# Security Policy

## Supported versions

Only the latest release receives fixes.

| Version | Supported |
| --- | --- |
| 1.0.x | Yes |

## Reporting a vulnerability

Please do not report security problems in public issues.

Use GitHub's private reporting instead: open the **Security** tab of this repository and choose **Report a vulnerability**. Include the SonarFix version, the steps to reproduce, and what an attacker could achieve.

You should get a first reply within a week. Once a fix is released, the report can be published with credit to you if you want.

## What SonarFix does and does not do

This helps you judge the impact of an issue:

- SonarFix runs as your user. The installer does not require administrator rights and installs to `%LOCALAPPDATA%\Programs\SonarFix`.
- It changes only Windows audio endpoint and session state, and records each change under `%LOCALAPPDATA%\SonarFix` so it can be undone.
- The only network access is the optional update check: an HTTPS request to the public GitHub Releases API of the official repository (`tuxp1/SonarFix`, fixed in the app). Redirects are not followed, responses are size-limited, and nothing is downloaded or run.
- A failed or unavailable update check never affects audio monitoring.
- Diagnostics are written to a file only when you export them. They are never uploaded.
- Release builds are not code-signed yet. Verify the SHA-256 checksum published with each release.

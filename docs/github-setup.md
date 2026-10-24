# GitHub repository and releases

Official repository: https://github.com/tuxp1/SonarFix (public, default branch `main`).

## Repository settings

- **Description:** A lightweight Windows utility that prevents duplicate microphone audio in Discord and lets you disable unwanted SteelSeries Sonar audio devices.
- **Topics:** `steelseries` `steelseries-sonar` `steelseries-gg` `discord` `microphone` `audio` `windows` `windows-audio`. Plain `sonar` mostly matches unrelated projects, and `open-source` adds nothing to search.
- **Features:** Issues on; Discussions optional; Wiki and Projects off.
- **Security:** keep *Private vulnerability reporting* enabled (SECURITY.md points to it).

## Update source

The app checks `tuxp1/SonarFix` (constant `OfficialRepository` in `manager/Preferences.cs`). Releases must be published there, as stable (not draft, not pre-release), with tags like `v1.0.0`.

## Publishing a release

1. Follow the release procedure in `HANDOFF.md` until `verify-release.ps1` prints PASS.
2. Commit and push the exact tree that was verified to `main`. `.gitignore` keeps `dist/`, `reports/`, `.tools/`, logs and local settings out; never commit build outputs.
3. Create tag `v<version>` on that commit and a release titled `SonarFix v<version>` with the body from `docs/releases/v<version>.md`.
4. Attach `dist/SonarFix-Setup-<version>.exe`, `dist/SonarFix-<version>-source.zip` and `dist/SHA256.json`.
5. Download the attachments again and compare their SHA-256 with `SHA256.json`.

## Screenshots

Real screenshots can be added to `docs/screenshots/` and linked from the Screenshots sections of both READMEs. Check them for personal data first.

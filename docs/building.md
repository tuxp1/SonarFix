# Build from source

[Back to README](../README.md)

## Requirements

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

## Build

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

## Project layout

| Path | Contents |
| --- | --- |
| `SonarFix-source/` | Go worker: audio monitoring, recovery journals, tests |
| `manager/` | Settings window (WPF, C#) and its self-tests |
| `installer/` | Inno Setup script, Arabic messages and logo images |
| `localization/strings.json` | All English and Arabic text, shared by the app and the installer |
| `VERSION` | The single version number used everywhere |

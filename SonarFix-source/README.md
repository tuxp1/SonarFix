# SonarFix worker

Go source of `SonarFix.exe`, the background worker that monitors and changes Windows audio state. It has no third-party Go modules.

Build it with `build.ps1` in the repository root, which also generates `version.go` and the Windows resources. See the main [README](../README.md) for requirements and the full build.

Live audio tests (`live_windows_test.go`, `render_trial_windows_test.go`) only run when explicitly enabled; a normal `go test` run uses fixtures and does not touch audio devices.

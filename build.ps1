param([string]$Go,[string]$ISCC,[string]$WinRes)
$ErrorActionPreference='Stop'
$root=$PSScriptRoot
if (-not $Go) { $Go=Join-Path $root '.tools\go\bin\go.exe'; if (-not (Test-Path -LiteralPath $Go)) {$Go=(Get-Command go -ErrorAction Stop).Source} }
if (-not $ISCC) { $ISCC=Join-Path $root '.tools\Inno\ISCC.exe'; if (-not (Test-Path -LiteralPath $ISCC)) {$ISCC=(Get-Command ISCC -ErrorAction Stop).Source} }
if (-not $WinRes) { $WinRes=Join-Path $root '.tools\bin\go-winres.exe'; if (-not (Test-Path -LiteralPath $WinRes)) {$WinRes=(Get-Command go-winres -ErrorAction Stop).Source} }
& (Join-Path $root 'generate-version.ps1')
$version=(Get-Content -LiteralPath (Join-Path $root 'VERSION') -Raw).Trim()
& (Join-Path $root 'generate-translations.ps1')
$dist=Join-Path $root 'dist'
$reports=Join-Path $root 'reports'
New-Item -ItemType Directory -Path $dist,$reports -Force | Out-Null
Push-Location (Join-Path $root 'SonarFix-source')
try {
    & $WinRes make --arch amd64
    if ($LASTEXITCODE -ne 0) {throw 'Windows resource build failed'}
    & $Go test -v . 2>&1 | Tee-Object (Join-Path $reports 'unit-tests.txt')
    if ($LASTEXITCODE -ne 0) {throw 'Go regression tests failed'}
    # Raw COM pointers from Windows produce four known unsafeptr diagnostics.
    # Save the full report and require every other vet analyzer to pass.
    # Windows PowerShell 5.1 turns native stderr into a terminating error under
    # 'Stop', so let this expected-to-warn run continue.
    $ErrorActionPreference='Continue'
    & $Go vet . 2>&1 | ForEach-Object {"$_"} | Set-Content (Join-Path $reports 'go-vet-full.txt')
    $ErrorActionPreference='Stop'
    # Create the report up front so a clean (silent) vet run still leaves a file.
    Set-Content (Join-Path $reports 'go-vet-other.txt') ''
    & $Go vet -unsafeptr=false . 2>&1 | Tee-Object (Join-Path $reports 'go-vet-other.txt')
    if ($LASTEXITCODE -ne 0) {throw 'Go vet failed'}
    & $Go build -trimpath -ldflags='-H=windowsgui' -o (Join-Path $dist 'SonarFix.exe') .
    if ($LASTEXITCODE -ne 0) {throw 'Worker build failed'}
} finally {Pop-Location}
$fx=Join-Path $env:WINDIR 'Microsoft.NET\Framework64\v4.0.30319'
$compiler=Join-Path $fx 'csc.exe'
& $compiler /nologo /target:winexe /platform:x64 /optimize+ ('/out:'+(Join-Path $dist 'SonarFix.Settings.exe')) ('/win32manifest:'+(Join-Path $root 'manager\app.manifest')) ('/win32icon:'+(Join-Path $root 'SonarFix-source\winres\icon.ico')) ('/resource:'+(Join-Path $root 'manager\Settings.xaml')+',SonarFix.Settings.xaml') ('/resource:'+(Join-Path $root 'localization\strings.json')+',SonarFix.strings.json') ('/resource:'+(Join-Path $root 'manager\Language.xaml')+',SonarFix.Language.xaml') /reference:System.Web.Extensions.dll /reference:System.Windows.Forms.dll /reference:System.Net.Http.dll /reference:System.Numerics.dll /reference:System.Drawing.dll ('/reference:'+$fx+'\WPF\PresentationFramework.dll') ('/reference:'+$fx+'\WPF\PresentationCore.dll') ('/reference:'+$fx+'\WPF\WindowsBase.dll') /reference:System.Xaml.dll (Join-Path $root 'manager\Settings.cs') (Join-Path $root 'manager\Version.cs') (Join-Path $root 'manager\SelfTests.cs') (Join-Path $root 'manager\Preferences.cs') (Join-Path $root 'manager\Updates.cs') (Join-Path $root 'manager\Tray.cs') (Join-Path $root 'manager\Health.cs') (Join-Path $root 'manager\AboutUpdates.cs') (Join-Path $root 'manager\FeatureTests.cs')
if ($LASTEXITCODE -ne 0) {throw 'Settings UI build failed'}
$ui=Start-Process -FilePath (Join-Path $dist 'SonarFix.Settings.exe') -ArgumentList '--self-test',('"'+(Join-Path $reports 'ui-localization.json')+'"') -WindowStyle Hidden -Wait -PassThru
if ($ui.ExitCode -ne 0) {throw 'Localization self-test failed'}
& $ISCC (Join-Path $root 'installer\SonarFix.iss') | Tee-Object (Join-Path $reports 'installer-build.txt')
if ($LASTEXITCODE -ne 0) {throw 'Installer build failed'}
Get-ChildItem -LiteralPath $dist -Filter '*.exe' | Get-FileHash -Algorithm SHA256 | Select-Object Hash,@{Name='File';Expression={Split-Path $_.Path -Leaf}} | ConvertTo-Json | Set-Content (Join-Path $dist 'SHA256.json')
Write-Output ('Built: '+(Join-Path $dist ('SonarFix-Setup-'+$version+'.exe')))








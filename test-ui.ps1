param([string]$Root=$PSScriptRoot)
$ErrorActionPreference='Stop'
function Snapshot {
    $settings=Join-Path $env:LOCALAPPDATA 'SonarFix\settings.json'
    $hash=if(Test-Path -LiteralPath $settings){(Get-FileHash -LiteralPath $settings).Hash}else{'absent'}
    try{$run=Get-ItemPropertyValue -LiteralPath 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Name 'SonarFix' -ErrorAction Stop}catch{$run=$null}
    try{$lang=Get-ItemPropertyValue -LiteralPath 'HKCU:\Software\SonarFix' -Name 'UILanguage' -ErrorAction Stop}catch{$lang=$null}
    $uiPath=Join-Path $env:LOCALAPPDATA 'SonarFix\ui-settings.json'
    $uiHash=if(Test-Path -LiteralPath $uiPath){(Get-FileHash -LiteralPath $uiPath).Hash}else{'absent'}
    return [ordered]@{UiSettingsHash=$uiHash;SettingsHash=$hash;Startup=$run;UILanguage=$lang;Workers=@(Get-Process -Name SonarFix -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Id)}
}
$version=(Get-Content -LiteralPath (Join-Path $Root 'VERSION') -Raw).Trim()
$before=Snapshot
$report=Join-Path $Root 'reports'
$ui=Start-Process -FilePath (Join-Path $Root 'dist\SonarFix.Settings.exe') -ArgumentList '--self-test',('"'+(Join-Path $report 'ui-localization.json')+'"') -WindowStyle Hidden -Wait -PassThru
if($ui.ExitCode -ne 0){throw 'Settings self-test failed'}
foreach($lang in @('en','ar')){
    $result=Join-Path $report ('installer-language-'+$lang+'.txt')
    if(Test-Path -LiteralPath $result){Remove-Item -LiteralPath $result}
    $log=Join-Path $report ('installer-validation-'+$lang+'.log')
    $p=Start-Process -FilePath (Join-Path $Root ('dist\SonarFix-Setup-'+$version+'.exe')) -ArgumentList ('/LANG='+$lang),'/VALIDATEONLY=1',('/UITESTFILE="'+$result+'"'),('/LOG="'+$log+'"'),'/SUPPRESSMSGBOXES' -WindowStyle Hidden -PassThru
    if(!$p.WaitForExit(20000)){throw ('Validation did not close: '+$p.Id)}
    # Exit 2 is deliberate cancellation before any installation/configuration.
    if($p.ExitCode -ne 2 -or !(Test-Path -LiteralPath $result) -or !((Get-Content -Raw $result).StartsWith('PASS:'))){throw ('Installer validation failed: '+$lang)}
}
$after=Snapshot
if(($before | ConvertTo-Json -Compress) -ne ($after | ConvertTo-Json -Compress)){throw 'UI tests changed persistent state'}
@{Result='PASS';SettingsUnchanged=$true;StartupUnchanged=$true;UILanguageUnchanged=$true;ReadOnly=$true;WorkerIDsUnchanged=$true} | ConvertTo-Json | Set-Content (Join-Path $report 'ui-state-preservation.json')
Write-Output 'PASS: application and installer UI tests; persistent settings, startup and language unchanged.'





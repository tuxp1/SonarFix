param([string]$Root=$PSScriptRoot)
$ErrorActionPreference='Stop'
$version=(Get-Content -LiteralPath (Join-Path $Root 'VERSION') -Raw).Trim()
$parts=@($version.Split('.'));while($parts.Count -lt 4){$parts+='0'}
$numeric=$parts -join '.'
$files=@('SonarFix.exe','SonarFix.Settings.exe',('SonarFix-Setup-'+$version+'.exe'))
$checks=foreach($file in $files){
    $v=(Get-Item -LiteralPath (Join-Path $Root ('dist\'+$file))).VersionInfo
    $actual=@($v.FileMajorPart,$v.FileMinorPart,$v.FileBuildPart,$v.FilePrivatePart) -join '.'
    if($actual -ne $numeric -or $v.ProductVersion.Trim() -ne $version){throw ('Version mismatch: '+$file)}
    [pscustomobject]@{File=$file;NumericVersion=$actual;ProductVersion=$v.ProductVersion.Trim();Result='PASS'}
}
$baseline=Get-Content -Raw -LiteralPath (Join-Path $Root 'tests\audio-core-baseline.json') | ConvertFrom-Json
foreach($entry in $baseline){if((Get-FileHash -LiteralPath (Join-Path $Root ('SonarFix-source\'+$entry.File))).Hash -ne $entry.Hash){throw ('Audio core modified: '+$entry.File)}}
$iss=Get-Content -Raw -LiteralPath (Join-Path $Root 'installer\SonarFix.iss')
if(!$iss.Contains('AppId=SonarFix') -or !$iss.Contains('#include "Version.iss"') -or !$iss.Contains('usAppMutexCheck') -or !$iss.Contains('--prepare-uninstall')){throw 'Installer identity/restore integration missing'}
$checks | ConvertTo-Json | Set-Content -Encoding UTF8 -LiteralPath (Join-Path $Root 'reports\release-integrity.json')
Write-Output ('PASS: product '+$version+', numeric '+$numeric+', four unchanged core audio files and installer recovery integration.')



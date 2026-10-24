param([string]$Root=$PSScriptRoot)
# Checks that the files in dist/ are the ones described by this source tree:
# installer checksum = release notes = SHA256.json, source ZIP = working tree,
# and the release decisions (license, notices, fixed update source) are present.
$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.IO.Compression,System.IO.Compression.FileSystem
$version=(Get-Content -LiteralPath (Join-Path $Root 'VERSION') -Raw).Trim()
$dist=Join-Path $Root 'dist'
$setup=Join-Path $dist ('SonarFix-Setup-'+$version+'.exe')
$zipPath=Join-Path $dist ('SonarFix-'+$version+'-source.zip')
$fail=[Collections.Generic.List[string]]::new()
function Need([bool]$ok,[string]$reason){if(!$ok){$fail.Add($reason)}}

# Installer checksum agrees everywhere it is published.
$hash=(Get-FileHash -LiteralPath $setup -Algorithm SHA256).Hash
$listed=@(Get-Content -Raw -LiteralPath (Join-Path $dist 'SHA256.json') | ConvertFrom-Json | ForEach-Object {$_})
Need (($listed | Where-Object {$_.File -eq (Split-Path $setup -Leaf)}).Hash -eq $hash) 'SHA256.json does not match the installer'
foreach($e in $listed){Need ((Get-FileHash -LiteralPath (Join-Path $dist $e.File)).Hash -eq $e.Hash) ('SHA256.json does not match '+$e.File)}
$notes=Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $Root ('docs\releases\v'+$version+'.md'))
Need ($notes.Contains($hash)) 'Release notes do not contain the installer SHA-256'
Need ((Get-Item $setup).LastWriteTimeUtc -ge (Get-Item (Join-Path $dist 'SonarFix.Settings.exe')).LastWriteTimeUtc) 'Installer is older than the built settings app'

# Source ZIP contains exactly the current files.
$zip=[IO.Compression.ZipFile]::OpenRead($zipPath)
try{
    $names=@{}
    foreach($entry in $zip.Entries){
        $relative=$entry.FullName.Substring('SonarFix/'.Length);$names[$relative]=$true
        $disk=Join-Path $Root $relative.Replace('/','\')
        if(!(Test-Path -LiteralPath $disk)){$fail.Add('ZIP has a file missing from the tree: '+$relative);continue}
        $sha=[Security.Cryptography.SHA256]::Create();$s=$entry.Open()
        try{$zipHash=[BitConverter]::ToString($sha.ComputeHash($s)).Replace('-','')}finally{$s.Dispose();$sha.Dispose()}
        Need ($zipHash -eq (Get-FileHash -LiteralPath $disk).Hash) ('ZIP copy differs from the tree: '+$relative)
    }
    foreach($required in @('LICENSE','THIRD_PARTY_NOTICES.md','README.md','README.ar.md','CONTRIBUTING.md','SECURITY.md','CHANGELOG.md','HANDOFF.md','TEST-REPORT.md','verify-release.ps1',
        '.gitignore','.github/pull_request_template.md','.github/ISSUE_TEMPLATE/bug_report.yml','.github/ISSUE_TEMPLATE/feature_request.yml','.github/ISSUE_TEMPLATE/config.yml',
        ('docs/releases/v'+$version+'.md'),'docs/github-setup.md','manager/AboutUpdates.cs','manager/Preferences.cs','installer/SonarFix.iss','localization/strings.json')){
        Need ($names.ContainsKey($required)) ('ZIP is missing '+$required)
    }
    Need (!$names.ContainsKey('updates.json')) 'ZIP still contains updates.json'
}finally{$zip.Dispose()}

# Release decisions.
Need ((Get-Content -LiteralPath (Join-Path $Root 'LICENSE'))[2] -eq 'Copyright (c) 2026 tuxp1') 'LICENSE copyright line'
Need ((Get-Content -Raw -LiteralPath (Join-Path $Root 'THIRD_PARTY_NOTICES.md')).Contains('generated with AI tools')) 'Notices lack the AI-generated icon statement'
$prefs=Get-Content -Raw -LiteralPath (Join-Path $Root 'manager\Preferences.cs')
$about=Get-Content -Raw -LiteralPath (Join-Path $Root 'manager\AboutUpdates.cs')
Need ($prefs.Contains('const string OfficialRepository="tuxp1/SonarFix";') -and $prefs.Contains('EffectiveRepository {get{return OfficialRepository;}}')) 'Update source is not fixed to tuxp1/SonarFix'
Need (!$prefs.Contains('updates.json') -and !$about.Contains('TextBox') -and !$about.Contains('SaveRepository')) 'Repository can still be changed'
Need (!(Test-Path -LiteralPath (Join-Path $Root 'updates.json'))) 'updates.json still exists'

if($fail.Count){$fail | ForEach-Object {Write-Output ('FAIL: '+$_)};throw 'Release verification failed'}
@{Result='PASS';Version=$version;InstallerSHA256=$hash;SourceZipSHA256=(Get-FileHash -LiteralPath $zipPath).Hash} | ConvertTo-Json | Set-Content -Encoding UTF8 -LiteralPath (Join-Path $Root 'reports\release-verification.json')
Write-Output ('PASS: installer '+$hash+' matches SHA256.json and release notes; source ZIP matches the tree; license, notices and fixed update source verified.')
